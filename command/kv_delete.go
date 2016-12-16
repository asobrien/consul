package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

// KVDeleteCommand is a Command implementation that is used to delete a key or
// prefix of keys from the key-value store.
type KVDeleteCommand struct {
	Ui cli.Ui
}

func (c *KVDeleteCommand) Help() string {
	helpText := `
Usage: consul kv delete [options] KEY_OR_PREFIX

  Removes the value from Consul's key-value store at the given path. If no
  key exists at the path, no action is taken.

  To delete the value for the key named "foo" in the key-value store:

      $ consul kv delete foo

  To delete all keys which start with "foo", specify the -recurse option:

      $ consul kv delete -recurse foo

  This will delete the keys named "foo", "food", and "foo/bar/zip" if they
  existed.

` + apiOptsText + `

KV Delete Options:

  -cas                    Perform a Check-And-Set operation. Specifying this
                          value also requires the -modify-index flag to be set.
                          The default value is false.

  -modify-index=<int>     Unsigned integer representing the ModifyIndex of the
                          key. This is used in combination with the -cas flag.

  -recurse                Recursively delete all keys with the path. The default
                          value is false.
`
	return strings.TrimSpace(helpText)
}

func (c *KVDeleteCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("get", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }
	datacenter := cmdFlags.String("datacenter", "", "")
	token := cmdFlags.String("token", "", "")
	cas := cmdFlags.Bool("cas", false, "")
	modifyIndex := cmdFlags.Uint64("modify-index", 0, "")
	recurse := cmdFlags.Bool("recurse", false, "")
	httpAddr := HTTPAddrFlag(cmdFlags)
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	key := ""

	// Check for arg validation
	args = cmdFlags.Args()
	switch len(args) {
	case 0:
		key = ""
	case 1:
		key = args[0]
	default:
		c.Ui.Error(fmt.Sprintf("Too many arguments (expected 1, got %d)", len(args)))
		return 1
	}

	// This is just a "nice" thing to do. Since pairs cannot start with a /, but
	// users will likely put "/" or "/foo", lets go ahead and strip that for them
	// here.
	if len(key) > 0 && key[0] == '/' {
		key = key[1:]
	}

	// If the key is empty and we are not doing a recursive delete, this is an
	// error.
	if key == "" && !*recurse {
		c.Ui.Error("Error! Missing KEY argument")
		return 1
	}

	// ModifyIndex is required for CAS
	if *cas && *modifyIndex == 0 {
		c.Ui.Error("Must specify -modify-index with -cas!")
		return 1
	}

	// Specifying a ModifyIndex for a non-CAS operation is not possible.
	if *modifyIndex != 0 && !*cas {
		c.Ui.Error("Cannot specify -modify-index without -cas!")
	}

	// It is not valid to use a CAS and recurse in the same call
	if *recurse && *cas {
		c.Ui.Error("Cannot specify both -cas and -recurse!")
		return 1
	}

	// Create and test the HTTP client
	conf := api.DefaultConfig()
	conf.Address = *httpAddr
	if conf.Token == "" {
		conf.Token = *token
	}
	client, err := api.NewClient(conf)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error connecting to Consul agent: %s", err))
		return 1
	}

	wo := &api.WriteOptions{
		Datacenter: *datacenter,
	}

	switch {
	case *recurse:
		if _, err := client.KV().DeleteTree(key, wo); err != nil {
			c.Ui.Error(fmt.Sprintf("Error! Did not delete prefix %s: %s", key, err))
			return 1
		}

		c.Ui.Info(fmt.Sprintf("Success! Deleted keys with prefix: %s", key))
		return 0
	case *cas:
		pair := &api.KVPair{
			Key:         key,
			ModifyIndex: *modifyIndex,
		}

		success, _, err := client.KV().DeleteCAS(pair, wo)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error! Did not delete key %s: %s", key, err))
			return 1
		}
		if !success {
			c.Ui.Error(fmt.Sprintf("Error! Did not delete key %s: CAS failed", key))
			return 1
		}

		c.Ui.Info(fmt.Sprintf("Success! Deleted key: %s", key))
		return 0
	default:
		if _, err := client.KV().Delete(key, wo); err != nil {
			c.Ui.Error(fmt.Sprintf("Error deleting key %s: %s", key, err))
			return 1
		}

		c.Ui.Info(fmt.Sprintf("Success! Deleted key: %s", key))
		return 0
	}
}

func (c *KVDeleteCommand) Synopsis() string {
	return "Removes data from the KV store"
}
