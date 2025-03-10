package cmd

import (
	"fmt"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/spf13/cobra"
	"path/filepath"
	"runtime"
	"strings"
)

// DebugNFSMountCmd implements the ddev debug nfsmount command
var DebugNFSMountCmd = &cobra.Command{
	Use:     "nfsmount",
	Short:   "Checks to see if nfs mounting works for current project",
	Example: "ddev debug nfsmount",
	Run: func(cmd *cobra.Command, args []string) {
		testVolume := "testnfsmount"
		containerName := "testnfscontainer"

		if len(args) != 0 {
			util.Failed("This command takes no additional arguments")
		}

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to debug nfsmount: %v", err)
		}
		oldContainer, err := dockerutil.FindContainerByName(containerName)
		if err == nil && oldContainer != nil {
			err = dockerutil.RemoveContainer(oldContainer.ID, 20)
			if err != nil {
				util.Failed("Failed to remove existing test container %s: %v", containerName, err)
			}
		}
		//nolint: errcheck
		dockerutil.RemoveVolume(testVolume)

		nfsServerAddr, err := dockerutil.GetNFSServerAddr()
		if err != nil {
			util.Failed("failed to GetNFSServerAddr(): %v", err)
		}
		shareDir := app.AppRoot
		// Workaround for Catalina sharing nfs as /System/Volumes/Data
		if runtime.GOOS == "darwin" && fileutil.IsDirectory(filepath.Join("/System/Volumes/Data", app.AppRoot)) {
			shareDir = filepath.Join("/System/Volumes/Data", app.AppRoot)
		}
		volume, err := dockerutil.CreateVolume(testVolume, "local", map[string]string{"type": "nfs", "o": fmt.Sprintf("addr=%s,hard,nolock,rw,wsize=32768,rsize=32768", nfsServerAddr), "device": ":" + dockerutil.MassageWindowsNFSMount(shareDir)}, nil)
		//nolint: errcheck
		defer dockerutil.RemoveVolume(testVolume)
		if err != nil {
			util.Failed("Failed to create volume %s: %v", testVolume, err)
		}
		_ = volume
		uidStr, _, _ := util.GetContainerUIDGid()

		_, out, err := dockerutil.RunSimpleContainer(versionconstants.GetWebImage(), containerName, []string{"sh", "-c", "findmnt -T /nfsmount && ls -d /nfsmount/.ddev"}, []string{}, []string{}, []string{"testnfsmount" + ":/nfsmount"}, uidStr, true, false, map[string]string{"com.ddev.site-name": ""})
		if err != nil {
			util.Warning("NFS does not seem to be set up yet, see debugging instructions at https://ddev.readthedocs.io/en/stable/users/install/performance/#debugging-ddev-start-failures-with-nfs_mount_enabled-true")
			util.Failed("Details: error=%v\noutput=%v", err, out)
		}
		output.UserOut.Printf(strings.TrimSpace(out))
		util.Success("")
		util.Success("Successfully accessed NFS mount of %s", app.AppRoot)
		switch {
		case app.NFSMountEnabledGlobal:
			util.Success("nfs_mount_enabled is set globally")
		case app.NFSMountEnabled:
			util.Success("nfs_mount_enabled is true in this project (%s), but is not set globally", app.Name)
		default:
			util.Warning("nfs_mount_enabled is not set either globally or in this project. \nUse `ddev config global --nfs-mount-enabled` to enable it.")
		}
	},
}

func init() {
	DebugCmd.AddCommand(DebugNFSMountCmd)
}
