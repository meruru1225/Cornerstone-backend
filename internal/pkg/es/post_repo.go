package es

type PostRepo interface {
}

type PostRepoImpl struct {
}

func NewPostRepo() PostRepo {
	return &PostRepoImpl{}
}
