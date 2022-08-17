//go:build mage
// +build mage

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	gitlab "github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-server/v6/model"
)

//To use this, unpack from https://github.com/magefile/mage/releases to $GOPATH/bin
//To build a static binary, run 'mage -compile make.exe'

var Default = Compile
var pluginSettings model.Manifest
var gitlabURL string
var gitlabToken string
var gitlabProjectID int
var bundleName string

type Hub mg.Namespace
type Lab mg.Namespace

func init() {
	gitlabURL = "https://gitlab.cdek.ru"
	gitlabToken = "glpat-AxBBC5XZJDdPZn9Ekv_T"
	gitlabProjectID = 1809
	jsonFile, err := os.Open("plugin.json")
	defer jsonFile.Close()
	if err == nil {
		jsonData, err := ioutil.ReadAll(jsonFile)
		if err == nil {
			err = json.Unmarshal(jsonData, &pluginSettings)
		}
	}
	if err == nil {
		fmt.Println("Settings for plugin <", pluginSettings.Id, "> loaded")
		bundleName = fmt.Sprintf("%s-%s.tar.gz", pluginSettings.Id, pluginSettings.Version)
	} else {
		fmt.Println("plugin.json fail:", err.Error())
	}
}

func (Lab) Release() error {
	git, err := gitlab.NewClient(gitlabToken, gitlab.WithBaseURL(gitlabURL+"/api/v4"))
	if err == nil {
		// https://gitlab.com/gitlab-org/release-cli/-/tree/master/docs/examples/release-assets-as-generic-package
		// https://gitlab.cdek.ru/help/user/packages/generic_packages/index.md#publish-a-package-file
		// https://pkg.go.dev/github.com/xanzy/go-gitlab#Client.UploadRequest
		packages, listPackagesResponse, err := git.Packages.ListProjectPackages(gitlabProjectID, &gitlab.ListProjectPackagesOptions{})
		if err == nil && listPackagesResponse.StatusCode == 200 {
			for i := range packages {
				fmt.Println("Checking package", packages[i].Name, "...")
				packageFiles, listFilesResponse, err := git.Packages.ListPackageFiles(gitlabProjectID, packages[i].ID, &gitlab.ListPackageFilesOptions{})
				if err == nil && listFilesResponse.StatusCode == 200 {
					for i := range packageFiles {
						fmt.Println(packageFiles[i].PackageID, packageFiles[i].ID, packageFiles[i].FileName)
						if packageFiles[i].FileName == bundleName {
							return fmt.Errorf("bundle named %s has already been published!", bundleName)
						}
					}
				}
			}
		} else {
			fmt.Println(listPackagesResponse.Status, err.Error())
		}
		//
		fmt.Println("Uploading bundle", bundleName)
		bundleFile, err := os.Open("./dist/" + bundleName)
		defer bundleFile.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
		publishedFile, publishResponse, err := git.GenericPackages.PublishPackageFile(
			gitlabProjectID,
			"Releases",
			pluginSettings.Version,
			bundleName,
			bundleFile,
			&gitlab.PublishPackageFileOptions{
				Select: gitlab.GenericPackageSelect(gitlab.SelectPackageFile),
				Status: gitlab.GenericPackageStatus(gitlab.PackageDefault),
			},
		)
		if err == nil && publishResponse.StatusCode == 200 {
			url1 := fmt.Sprintf("https://gitlab.cdek.ru/FrontDev/mm-pliugin-video/-/package_files/%d/download", publishedFile.ID)
			fmt.Println(url1)
			url2 := gitlabURL + publishedFile.File.URL
			fmt.Println(url2)
			bundleLink := &gitlab.ReleaseAssetLinkOptions{
				Name:     gitlab.String("get bundle"),
				URL:      gitlab.String(url2),
				LinkType: gitlab.LinkType(gitlab.PackageLinkType),
			}
			year, month, day := time.Now().Date()
			releaseName := fmt.Sprintf("Released on %v %d, %d", month, day, year)
			release, releaseResponse, err := git.Releases.CreateRelease(
				gitlabProjectID,
				&gitlab.CreateReleaseOptions{
					Ref:         gitlab.String("master"), // It can be a commit SHA, another tag name, or a branch name.
					Name:        gitlab.String(releaseName),
					TagName:     gitlab.String("v" + pluginSettings.Version),
					TagMessage:  gitlab.String(""),
					Description: gitlab.String("no description provided (for quick releases)"),
					ReleasedAt:  gitlab.Time(time.Now()),
					Assets: &gitlab.ReleaseAssetsOptions{
						Links: []*gitlab.ReleaseAssetLinkOptions{bundleLink},
					},
				},
			)
			if err == nil && releaseResponse.StatusCode == 201 {
				fmt.Println(releaseResponse.Status, release.Name, "released")
			} else {
				fmt.Println(releaseResponse.Status)
				fmt.Println(err.Error())
			}
		} else {
			fmt.Println(publishResponse.Status)
			fmt.Println(err.Error())
		}
	}
	return err
}

func (Lab) ListProjects() error {
	git, err := gitlab.NewClient(gitlabToken, gitlab.WithBaseURL(gitlabURL))
	if err == nil {
		name := "video"
		fmt.Println("Searching GitLab for", name, "...")
		projectsOptions := &gitlab.ListProjectsOptions{Search: gitlab.String(name)}
		projects, _, err := git.Projects.ListProjects(projectsOptions)
		if err == nil {
			for i := range projects {
				fmt.Println(projects[i].ID, projects[i].WebURL)
			}
		}
	}
	return err
}

func Install() error {
	err := Build()
	if err == nil {
		return Deploy()
	}
	return err
}

func Deploy() error {
	// mg.Deps(Build)
	// siteURL := os.Getenv("MM_SITEURL")
	// adminToken := os.Getenv("MM_ADMIN_TOKEN")
	adminUsername := "Denis"
	adminPassword := "##332211qqwweE"
	siteURL := "https://dev-talk.cdek.ru"
	client := model.NewAPIv4Client(siteURL)
	// client.SetToken(adminToken)
	fmt.Printf("Authenticating as %s against %s: ", adminUsername, siteURL)
	_, _, err := client.Login(adminUsername, adminPassword)
	if err == nil {
		fmt.Printf("Ok\n")
	} else {
		fmt.Printf("fail!\n")
		return err
	}
	bundlePath := fmt.Sprintf("./dist/%s", bundleName)
	pluginBundle, err := os.Open(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", bundlePath, err)
	}
	defer pluginBundle.Close()

	fmt.Print("Uploading plugin via API: ")
	_, _, err = client.UploadPluginForced(pluginBundle)
	if err == nil {
		fmt.Printf("Ok\n")
	} else {
		fmt.Printf("fail!\n")
		return fmt.Errorf("failed to upload plugin bundle: %s", err.Error())
	}

	fmt.Print("Enabling plugin: ")
	_, err = client.EnablePlugin(pluginSettings.Id)
	if err == nil {
		fmt.Printf("Ok\n")
	} else {
		fmt.Printf("fail!\n")
		return err
	}
	return nil
}

func Build() error {
	mg.Deps(Compile)
	destinationDir := "./dist/" + pluginSettings.Id
	os.MkdirAll(destinationDir, 0755)
	os.MkdirAll(destinationDir+"/webapp", 0755)
	err := copyFile("plugin.json", destinationDir)
	err = copyFile("webapp/dist/main.js", destinationDir+"/webapp")
	err = copyDir("assets", destinationDir)
	err = run("./dist", "tar", "-cvzf", bundleName, pluginSettings.Id)
	return err
}

func Compile() error {
	fmt.Println("building...")

	sh.Rm("dist")
	if err := os.MkdirAll("dist", 0755); err != nil {
		return err
	}
	os.MkdirAll("dist/server", 0755)
	os.MkdirAll("dist/webapp", 0755)

	for tag := range pluginSettings.Server.Executables {
		fmt.Println("making server executable for", tag, "...")
		splitted := strings.Split(tag, "-")
		output := fmt.Sprintf("../dist/%s/server/plugin-%s", pluginSettings.Id, tag)
		cmd := exec.Command("go", "build", "-trimpath", "-o", output)
		cmd.Env = []string{
			"GOOS=" + splitted[0],
			"GOARCH=" + splitted[1],
			"PATH=" + os.Getenv("PATH"),
			"HOME=" + os.Getenv("HOME"),
			"GOPATH=" + os.Getenv("GOPATH"),
			"GO111MODULE=on",
		}
		if runtime.GOOS == "windows" {
			cmd.Env = append(cmd.Env,
				"GOCACHE="+os.Getenv("LOCALAPPDATA")+"\\go-build",
				"GOTMPDIR="+os.Getenv("LOCALAPPDATA")+"\\Temp",
			)
		}
		cmd.Dir = "./server"
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = "./webapp"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return sh.Run("go", "version")
}

func SetUp() error {
	if err := sh.Run("go", "mod", "download"); err != nil {
		return err
	}
	err := run("./webapp", "npm", "install", "--legacy-peer-deps") // npm install moment --save-dev  --legacy-peer-deps
	return err
}

func download(url string, path []string) error {
	fileName := filepath.Join(path...)
	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	return nil
}

func copyFile(src, dst string) error {
	if runtime.GOOS == "windows" {
		return run(".", "copy", src, dst)
	} else {
		return run(".", "cp", src, dst)
	}
}

func copyDir(src, dst string) error {
	var err error
	if runtime.GOOS == "windows" {
		err = run(".", "copy", src, dst)
	} else {
		err = run(".", "cp", "-r", src, dst)
	}
	return err
}

func run(dir, exe string, args ...string) error {
	cmd := exec.Command(exe, args...)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
	}
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
