//go:build mage
// +build mage

package main

import (
	"bytes"
	"context"
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

	"github.com/google/go-github/v45/github"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	gitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-server/v6/model"
)

//To use this, unpack from https://github.com/magefile/mage/releases to $GOPATH/bin
//To build a static binary, run 'mage -compile make.exe'

type serverCredentials struct {
	URL         string
	AccessToken string
	ID          interface{}
}

type serverConfigurations struct {
	Mattermost serverCredentials
	GitHub     serverCredentials
	GitLab     serverCredentials
}

var Default = Compile
var pluginSettings model.Manifest
var configuration serverConfigurations
var bundleName string
var standaloneName string

type Hub mg.Namespace
type Lab mg.Namespace

func init() {
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
		standaloneName = fmt.Sprintf("%s-%s-standalone.tar.gz", pluginSettings.Id, pluginSettings.Version)
	} else {
		fmt.Println("plugin.json fail:", err.Error())
	}
	jsonFile, err = os.Open("servers.json")
	if err == nil {
		defer jsonFile.Close()
		jsonData, err := ioutil.ReadAll(jsonFile)
		if err == nil {
			err = json.Unmarshal(jsonData, &configuration)
		}
	} else {
		fmt.Println("servers.json not found...")
		srv := serverCredentials{URL: "https://", ID: "n\\a", AccessToken: "n\\a"}
		cfg := serverConfigurations{Mattermost: srv, GitHub: srv, GitLab: srv}
		jsonData, _ := json.Marshal(cfg)
		pretty := &bytes.Buffer{}
		json.Indent(pretty, jsonData, "", "   ")
		err = ioutil.WriteFile("servers.json", pretty.Bytes(), os.ModePerm)
		if err == nil {
			fmt.Println("servers.json created")
		} else {
			fmt.Println(err)
		}
	}
}

func (Hub) Release() error {
	ctx := context.Background()
	splitted := strings.Split(configuration.GitHub.ID.(string), "/")
	owner := splitted[0]
	repo := splitted[1]
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: configuration.GitHub.AccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	fmt.Println(client.BaseURL.User)
	releases, listResponse, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{})
	if err == nil {
		for i := range releases {
			fmt.Println("Checking release", *releases[i].ID)
			assets, assetsResponse, err := client.Repositories.ListReleaseAssets(ctx, owner, repo, *releases[i].ID, &github.ListOptions{})
			if err == nil {
				for i := range assets {
					fmt.Println(*assets[i].Name)
				}
			} else {
				fmt.Println(assetsResponse.Status)
			}
		}
	} else {
		fmt.Println(listResponse.Status)
		return err
	}
	year, month, day := time.Now().Date()
	releaseName := fmt.Sprintf("Released on %v %d, %d", month, day, year)
	release := &github.RepositoryRelease{
		TagName:              github.String("v" + pluginSettings.Version),
		Name:                 github.String(releaseName),
		Body:                 github.String("no description provided (for quick releases)"),
		GenerateReleaseNotes: github.Bool(false),
	}
	createdRelease, releaseResponse, err := client.Repositories.CreateRelease(ctx, owner, repo, release)
	if err == nil {
		fmt.Println("New release created: id =", *createdRelease.ID, ", name =", *createdRelease.Name)
		fmt.Printf("Uploading bundle %s ...", bundleName)
		bundleFile, err := os.Open("./dist/" + bundleName)
		if err == nil {
			defer bundleFile.Close()
		} else {
			fmt.Println(err.Error())
			return err
		}
		uploadedAsset, uploadResponse, err := client.Repositories.UploadReleaseAsset(
			ctx, owner, repo, *createdRelease.ID,
			&github.UploadOptions{Name: bundleName},
			bundleFile,
		)
		if err == nil {
			fmt.Printf("Ok\nnew url: %s\n", *uploadedAsset.URL)
			return nil
		} else {
			fmt.Printf("fail!\n")
			fmt.Println(uploadResponse.Status)
			return err
		}
	} else {
		fmt.Println(releaseResponse.Status)
	}
	return err
}

func (Lab) Release() error {
	git, err := gitlab.NewClient(configuration.GitLab.AccessToken, gitlab.WithBaseURL(configuration.GitLab.URL+"/api/v4"))
	if err == nil {
		project, projectResponse, err := git.Projects.GetProject(configuration.GitLab.ID, &gitlab.GetProjectOptions{})
		if err != nil || projectResponse.StatusCode != 200 {
			fmt.Println(projectResponse.Status)
			return err
		}
		packages, listPackagesResponse, err := git.Packages.ListProjectPackages(project.ID, &gitlab.ListProjectPackagesOptions{})
		if err == nil && listPackagesResponse.StatusCode == 200 {
			for i := range packages {
				fmt.Println("Checking package", packages[i].Name, "...")
				packageFiles, listFilesResponse, err := git.Packages.ListPackageFiles(project.ID, packages[i].ID, &gitlab.ListPackageFilesOptions{})
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
			project.ID,
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
			url1 := fmt.Sprintf("%s/-/package_files/%d/download", project.WebURL, publishedFile.ID)
			fmt.Println(url1)
			url2 := configuration.GitLab.URL + publishedFile.File.URL
			fmt.Println(url2)
			bundleLink := &gitlab.ReleaseAssetLinkOptions{
				Name:     gitlab.String("get bundle"),
				URL:      gitlab.String(url1),
				LinkType: gitlab.LinkType(gitlab.PackageLinkType),
			}
			year, month, day := time.Now().Date()
			releaseName := fmt.Sprintf("Released on %v %d, %d", month, day, year)
			release, releaseResponse, err := git.Releases.CreateRelease(
				project.ID,
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
				fmt.Println(releaseResponse.Status, "'", release.Name, "'")
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

func (Lab) Standalone() error {
	output := fmt.Sprintf("../dist/standalone/readonly_channels-%s", pluginSettings.Version)
	cmd := exec.Command("go", "build", "-trimpath", "-o", output)
	cmd.Env = []string{
		"GOOS=linux",
		"GOARCH=amd64",
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"GOPATH=" + os.Getenv("GOPATH"),
		// "GO111MODULE=on",
	}
	if runtime.GOOS == "windows" {
		cmd.Env = append(cmd.Env,
			"GOCACHE="+os.Getenv("LOCALAPPDATA")+"\\go-build",
			"GOTMPDIR="+os.Getenv("LOCALAPPDATA")+"\\Temp",
		)
	}
	cmd.Dir = "./standalone"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	err := run("./dist/standalone", "tar", "-cvzf", "../"+standaloneName, "readonly_channels-"+pluginSettings.Version)
	git, err := gitlab.NewClient(configuration.GitLab.AccessToken, gitlab.WithBaseURL(configuration.GitLab.URL+"/api/v4"))
	if err == nil {
		pidCast := configuration.GitLab.ID.(float64)
		project, projectResponse, err := git.Projects.GetProject(int(pidCast), &gitlab.GetProjectOptions{})
		if err != nil {
			fmt.Println(projectResponse.Status)
			return err
		}
		fmt.Println("GitLab project", project.NameWithNamespace, "obtained")
		fmt.Println("Uploading package", standaloneName)
		bundleFile, err := os.Open("./dist/" + standaloneName)
		defer bundleFile.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
		publishedFile, publishResponse, err := git.GenericPackages.PublishPackageFile(
			project.ID,
			"Releases",
			pluginSettings.Version,
			standaloneName,
			bundleFile,
			&gitlab.PublishPackageFileOptions{
				Select: gitlab.GenericPackageSelect(gitlab.SelectPackageFile),
				Status: gitlab.GenericPackageStatus(gitlab.PackageDefault),
			},
		)
		if err == nil {
			fmt.Println(publishedFile.ID)
		} else {
			fmt.Println(publishResponse.Status)
			return err
		}
	}
	return err
}

func (Lab) ListProjects() error {
	git, err := gitlab.NewClient(configuration.GitLab.AccessToken, gitlab.WithBaseURL(configuration.GitLab.URL))
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

func Logs() error {
	fmt.Printf("Listing logs from %s with <%s> substring: \n", configuration.Mattermost.URL, pluginSettings.Id)
	client := model.NewAPIv4Client(configuration.Mattermost.URL)
	client.SetToken(configuration.Mattermost.AccessToken)
	logsPerPage := 1000
	for i := 0; i < 100; i++ {
		fmt.Printf("Getting page %d: ", i)
		page, pageResponse, err := client.GetLogs(i, logsPerPage)
		if err == nil {
			fmt.Printf("%d \n", pageResponse.StatusCode)
			for j := range page {
				found := strings.Contains(page[j], pluginSettings.Id)
				if found {
					fmt.Println(page[j])
				}
			}
			if len(page) < logsPerPage {
				fmt.Println("Breaking at", i)
				continue
			}
		} else {
			fmt.Printf("%s \n", err.Error())
		}
	}
	return nil
}

func Deploy() error {
	fmt.Printf("Deploying to %s: \n", configuration.Mattermost.URL)
	client := model.NewAPIv4Client(configuration.Mattermost.URL)
	client.SetToken(configuration.Mattermost.AccessToken)
	// _, _, err := client.Login(adminUsername, adminPassword)
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
		//Windows fix for setting executable bit
		if runtime.GOOS == "windows" {
			err := run(cmd.Dir, "git", "update-index", "--chmod=+x", output)
			if err != nil {
				return err
			}
		}
	}

	return run("./webapp", "npm", "run", "build")
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
		var do = strings.NewReplacer("/", "\\")
		return run(".", "cmd", "/C", "copy", do.Replace(src), do.Replace(dst))
	} else {
		return run(".", "cp", src, dst)
	}
}

func copyDir(src, dst string) error {
	var err error
	if runtime.GOOS == "windows" {
		var do = strings.NewReplacer("/", "\\")
		dst = dst + "/" + src
		err = run(".", "cmd", "/C", "xcopy", "/I", do.Replace(src), do.Replace(dst))
	} else {
		err = run(".", "cp", "-r", src, dst)
	}
	return err
}

func run(dir, exe string, args ...string) error {
	cmd := exec.Command(exe, args...)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
	}
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
