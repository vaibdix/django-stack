package render

import (
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
)

func Has(addOn config.AddOn) func(config.Spec) bool {
	return func(c config.Spec) bool { return c.Has(addOn) }
}

func (p Project) composeFiles() (FileSet, error) {
	files := FileSet{}
	if !p.Cfg.Database().UsesDocker() {
		return files, nil
	}
	content, err := dbdocker.Compose(p.Cfg.Name().String(), p.Cfg.Database())
	if err != nil {
		return nil, err
	}
	files.add(dbdocker.ComposeFile, content, FileMode)
	return files, nil
}
