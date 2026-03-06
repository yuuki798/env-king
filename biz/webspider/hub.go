package webspider

type Hub struct {
	DockerMirrorListHandler *DockerMirrorListHandler
}

func NewHub() *Hub {
	return &Hub{
		DockerMirrorListHandler: NewDockerMirrorListHandler(""),
	}
}
