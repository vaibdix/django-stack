package config

import "github.com/vaibdix/django-stack/internal/catalog"

func Tools() catalog.Tools { return catalog.Load().Tools }

func NodeMinimum() int { return catalog.Load().MinimumMajor("node") }

func (s Spec) HostNeeds() []catalog.Need {
	var needs []catalog.Need
	for _, rule := range catalog.Load().Host {
		if !s.requires(rule) {
			continue
		}
		for _, text := range rule.Needs {
			need, err := catalog.ParseNeed(text)
			if err == nil {
				needs = append(needs, need)
			}
		}
	}
	return needs
}

func (s Spec) requires(rule catalog.HostRule) bool {
	kind, name := rule.Condition()
	switch kind {
	case "addon":
		return s.Has(AddOn(name))
	case "stack":
		return s.stack.Has(name)
	case "database":
		return s.database.UsesDocker()
	default:
		return false
	}
}

func StackNeeds(stack Stack) []catalog.Need { return needsWhere("stack", stack.Has) }

func AddOnNeeds(addOn AddOn) []catalog.Need {
	return needsWhere("addon", func(name string) bool { return AddOn(name) == addOn })
}

func needsWhere(kind string, holds func(name string) bool) []catalog.Need {
	var needs []catalog.Need
	for _, rule := range catalog.Load().Host {
		ruleKind, name := rule.Condition()
		if ruleKind != kind || !holds(name) {
			continue
		}
		for _, text := range rule.Needs {
			if need, err := catalog.ParseNeed(text); err == nil {
				needs = append(needs, need)
			}
		}
	}
	return needs
}
