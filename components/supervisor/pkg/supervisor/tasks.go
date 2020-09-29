// Copyright (c) 2020 TypeFox GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package supervisor

import (
	"context"
	"strings"
	"sync"

	"github.com/gitpod-io/gitpod/common-go/log"
	"github.com/gitpod-io/gitpod/supervisor/api"
	"github.com/gitpod-io/gitpod/supervisor/pkg/terminal"
)

type tasksManager struct {
	config          *Config
	tasks           map[string]*api.TasksStatus
	ready           chan struct{}
	terminalService *terminal.MuxTerminalService
}

func newTasksManager(config *Config, terminalService *terminal.MuxTerminalService) *tasksManager {
	return &tasksManager{
		config:          config,
		terminalService: terminalService,
		tasks:           make(map[string]*api.TasksStatus),
		ready:           make(chan struct{}),
	}
}

func (tm *tasksManager) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(tm.ready)

	tasks, err := tm.config.getGitpodTasks()
	if err != nil {
		log.WithError(err).Fatal()
		return
	}
	if tasks == nil {
		log.Info("no gitpod tasks found")
		return
	}
	for _, task := range *tasks {
		command := ""
		for _, maybeCmd := range []*string{task.Before, task.Init, task.Command} {
			if maybeCmd != nil {
				cmd := strings.TrimSpace(*maybeCmd)
				if cmd != "" {
					if command != "" {
						command += " && "
					}
					command += "{\n" + cmd + "\n}"
				}
			}
		}
		if command == "" {
			continue
		}
		taskLog := log.WithField("command", command)
		taskLog.Info("starting a task terminal...")
		resp, err := tm.terminalService.Open(ctx, &api.OpenTerminalRequest{})
		if err != nil {
			taskLog.WithError(err).Fatal("cannot open new task terminal")
			continue
		}

		alias := resp.Alias
		taskLog = taskLog.WithField("alias", alias)
		_, err = tm.terminalService.Write(ctx, &api.WriteTerminalRequest{
			Alias: alias,
			Stdin: []byte(command),
		})
		if err != nil {
			taskLog.WithError(err).Fatal("cannot send a command to a task terminal")
			err = tm.terminalService.Mux.Close(alias)
			if err != nil {
				taskLog.WithError(err).Fatal("cannot close a task terminal")
			}
			continue
		}

		tm.tasks[alias] = &api.TasksStatus{
			Alias: alias,
		}
		taskLog.Info("task terminal has been started")
	}
}
