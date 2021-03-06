package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/mitchellh/cli"
)

// SnapshotRestoreCommand is a Command implementation that is used to restore
// the state of the Consul servers for disaster recovery.
type SnapshotRestoreCommand struct {
	Ui cli.Ui
}

func (c *SnapshotRestoreCommand) Help() string {
	helpText := `
Usage: consul snapshot restore [options] FILE

  Restores an atomic, point-in-time snapshot of the state of the Consul servers
  which includes key/value entries, service catalog, prepared queries, sessions,
  and ACLs.

  Restores involve a potentially dangerous low-level Raft operation that is not
  designed to handle server failures during a restore. This command is primarily
  intended to be used when recovering from a disaster, restoring into a fresh
  cluster of Consul servers.

  If ACLs are enabled, a management token must be supplied in order to perform
  snapshot operations.

  To restore a snapshot from the file "backup.snap":

    $ consul snapshot restore backup.snap

  For a full list of options and examples, please see the Consul documentation.

` + apiOptsText

	return strings.TrimSpace(helpText)
}

func (c *SnapshotRestoreCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("get", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }
	datacenter := cmdFlags.String("datacenter", "", "")
	token := cmdFlags.String("token", "", "")
	httpAddr := HTTPAddrFlag(cmdFlags)
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var file string

	args = cmdFlags.Args()
	switch len(args) {
	case 0:
		c.Ui.Error("Missing FILE argument")
		return 1
	case 1:
		file = args[0]
	default:
		c.Ui.Error(fmt.Sprintf("Too many arguments (expected 1, got %d)", len(args)))
		return 1
	}

	// Create and test the HTTP client
	conf := api.DefaultConfig()
	conf.Datacenter = *datacenter
	conf.Address = *httpAddr
	if *token != "" {
		conf.Token = *token
	}
	client, err := api.NewClient(conf)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error connecting to Consul agent: %s", err))
		return 1
	}

	// Open the file.
	f, err := os.Open(file)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error opening snapshot file: %s", err))
		return 1
	}
	defer f.Close()

	// Restore the snapshot.
	err = client.Snapshot().Restore(nil, f)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error restoring snapshot: %s", err))
		return 1
	}

	c.Ui.Info("Restored snapshot")
	return 0
}

func (c *SnapshotRestoreCommand) Synopsis() string {
	return "Restores snapshot of Consul server state"
}
