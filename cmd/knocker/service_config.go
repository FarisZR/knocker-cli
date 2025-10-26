package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	internalService "github.com/FarisZR/knocker-cli/internal/service"
	"github.com/kardianos/service"
	"github.com/spf13/viper"
)

const (
	serviceName        = "knocker"
	serviceDisplayName = "Knocker IP Whitelist Service"
	serviceDescription = "Automatically whitelists the external IP of this device."
)

func newServiceInstance(includeExecutable bool) (service.Service, error) {
	cfg, err := newServiceConfig(includeExecutable)
	if err != nil {
		return nil, err
	}

	return service.New(&program{}, cfg)
}

func newServiceConfig(includeExecutable bool) (*service.Config, error) {
	cfg := &service.Config{
		Name:        serviceName,
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
		Option:      service.KeyValue{},
		Dependencies: []string{
			"After=network-online.target",
			"Wants=network-online.target",
		},
	}

	if includeExecutable {
		executable, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("could not determine executable path: %w", err)
		}

		cfg.Executable = executable
		cfg.WorkingDirectory = filepath.Dir(executable)
	}

	applyPlatformServiceOptions(cfg)

	return cfg, nil
}

func applyPlatformServiceOptions(cfg *service.Config) {
	if cfg.Option == nil {
		cfg.Option = service.KeyValue{}
	}

	ttl := viper.GetInt("ttl")
	restartDelay := internalService.RestartDelay(ttl)
	restartSeconds := int(restartDelay / time.Second)
	if restartSeconds < 1 {
		restartSeconds = 1
	}

	switch runtime.GOOS {
	case "linux":
		cfg.Option["UserService"] = true
		cfg.Option["Restart"] = "always"
		cfg.Option["SystemdScript"] = systemdUserUnitTemplate(restartSeconds)
	case "darwin":
		cfg.Option["UserService"] = true
		cfg.Option["RunAtLoad"] = true
		cfg.Option["SessionCreate"] = true
		cfg.Option["KeepAlive"] = true
	default:
		cfg.Option["Restart"] = "always"
	}
}

func systemdUserUnitTemplate(restartSec int) string {
	if restartSec <= 0 {
		restartSec = 30
	}

	return fmt.Sprintf(systemdUnitTemplate, restartSec)
}

const systemdUnitTemplate = `[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}
{{range $i, $dep := .Dependencies}}
{{$dep}}
{{end}}

[Service]
StartLimitInterval=5
StartLimitBurst=10
ExecStart={{.Path|cmdEscape}}{{range .Arguments}} {{.|cmd}}{{end}}
{{if .ChRoot}}RootDirectory={{.ChRoot|cmd}}{{end}}
{{if .WorkingDirectory}}WorkingDirectory={{.WorkingDirectory|cmdEscape}}{{end}}
{{if .UserName}}User={{.UserName}}{{end}}
{{if .ReloadSignal}}ExecReload=/bin/kill -{{.ReloadSignal}} "$MAINPID"{{end}}
{{if .PIDFile}}PIDFile={{.PIDFile|cmd}}{{end}}
{{if and .LogOutput .HasOutputFileSupport -}}
StandardOutput=file:{{.LogDirectory}}/{{.Name}}.out
StandardError=file:{{.LogDirectory}}/{{.Name}}.err
{{- end}}
{{if gt .LimitNOFILE -1 }}LimitNOFILE={{.LimitNOFILE}}{{end}}
{{if .Restart}}Restart={{.Restart}}{{else}}Restart=always{{end}}
{{if .SuccessExitStatus}}SuccessExitStatus={{.SuccessExitStatus}}{{end}}
RestartSec=%d
EnvironmentFile=-%%h/.config/knocker/env
{{range $k, $v := .EnvVars -}}
Environment={{$k}}={{$v}}
{{end -}}

[Install]
WantedBy=default.target
`
