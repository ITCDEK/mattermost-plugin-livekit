package main

import (
	"reflect"

	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.

type configuration struct {
	Secure     bool
	Host       string
	Port       int
	TurnSecure bool
	TurnHost   string
	TurnPort   int
	TurnUDP    int
	ApiKey     string `json:"-"`
	ApiValue   string `json:"-"`
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *configuration) Clone() *configuration {
	var clone = *c
	return &clone
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (lkp *LiveKitPlugin) getConfiguration() *configuration {
	lkp.configurationLock.RLock()
	defer lkp.configurationLock.RUnlock()

	if lkp.configuration == nil {
		return &configuration{}
	}

	return lkp.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (lkp *LiveKitPlugin) setConfiguration(configuration *configuration) {
	lkp.configurationLock.Lock()
	defer lkp.configurationLock.Unlock()

	if configuration != nil && lkp.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	lkp.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (lkp *LiveKitPlugin) OnConfigurationChange() error {
	var configuration = new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	err := lkp.API.LoadPluginConfiguration(configuration)
	if err == nil {
		lkp.setConfiguration(configuration)
		return nil
	} else {
		return errors.Wrap(err, "failed to load plugin configuration")
	}
}
