// Copyright (c) 2020 TypeFox GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gitpod-io/gitpod/supervisor/api"
	"github.com/spf13/cobra"
)

var newuidmapCmdOpts = struct {
	GID bool
}{}

var newuidmapCmd = &cobra.Command{
	Use:    "newuidmap <pid> <inContainerID> <hostID> <size> ... [<inContainerID> <hostID> <size>]",
	Short:  "establishes a new UID mapping for a user-namespace",
	Hidden: true,
	Args:   cobra.MinimumNArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn := dialSupervisor()
		client := api.NewControlServiceClient(conn)

		pid, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse PID: %w", err)
		}

		mapping := make([]*api.NewuidmapRequest_Mapping, 0, (len(args)-1)/3)
		for i := 1; i < len(args); i++ {
			icid, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return fmt.Errorf("cannot parse inContainerID (arg %d): %w", i, err)
			}
			i++

			hid, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return fmt.Errorf("cannot parse inContainerID (arg %d): %w", i, err)
			}
			i++

			sze, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil {
				return fmt.Errorf("cannot parse inContainerID (arg %d): %w", i, err)
			}

			mapping = append(mapping, &api.NewuidmapRequest_Mapping{
				ContainerId: icid,
				HostId:      hid,
				Size:        sze,
			})
		}

		if (len(args)-1)%3 != 0 {
			return fmt.Errorf("arguments must be tripples")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, err = client.Newuidmap(ctx, &api.NewuidmapRequest{
			Pid:     pid,
			Gid:     newuidmapCmdOpts.GID,
			Mapping: mapping,
		})
		return err
	},
}

func init() {
	rootCmd.AddCommand(newuidmapCmd)

	newuidmapCmd.Flags().BoolVarP(&newuidmapCmdOpts.GID, "gid", "g", false, "create GID mapping rather than UID")
}
