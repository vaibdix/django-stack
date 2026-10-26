package scaffold

import "github.com/vaibdix/django-stack/internal/config"

func pydanticAddOn() addOn {
	return addOn{
		enabled: has(config.Pydantic),
		contribute: func(*Env) contribution {
			return contribution{readme: func(d *readmeData) { d.Pydantic = true }}
		},
	}
}
