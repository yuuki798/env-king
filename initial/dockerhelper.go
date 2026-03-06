package initial

import (
	"yuuki798/env-king/biz/dockerhelper"
	"yuuki798/env-king/i"
)

func InitDockerHelperHub() {
	i.DockerHelperHub = dockerhelper.NewHub()
}
