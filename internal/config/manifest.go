package config

import (
	"encoding/json"
	"fmt"
)

const ManifestSchema = 1

func migrateManifest(doc map[string]json.RawMessage) error {
	schema := 0
	if raw, ok := doc["schemaVersion"]; ok {
		if err := json.Unmarshal(raw, &schema); err != nil {
			return fmt.Errorf("invalid manifest: schemaVersion: %w", err)
		}
	}
	if schema > ManifestSchema {
		return fmt.Errorf("this manifest has schema %d, and this django-stack reads up to schema %d; update django-stack to replay it", schema, ManifestSchema)
	}
	for ; schema < ManifestSchema; schema++ {
		if err := migrateManifestFrom(schema, doc); err != nil {
			return fmt.Errorf("migrate the manifest from schema %d: %w", schema, err)
		}
	}
	return nil
}

func migrateManifestFrom(schema int, doc map[string]json.RawMessage) error {
	switch schema {
	case 0:
		doc["schemaVersion"] = json.RawMessage("1")
		return nil
	default:
		return fmt.Errorf("no migration from schema %d", schema)
	}
}
