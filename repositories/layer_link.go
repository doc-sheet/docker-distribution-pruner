package repositories

import "gitlab.com/gitlab-org/docker-distribution-pruner/digest"

type LayerLink struct {
	repository *Repository
	Manifest   digest.Digest
	Used       int
}

func (l *LayerLink) Path() string {

}
