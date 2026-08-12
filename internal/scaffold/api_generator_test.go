package scaffold

func generateBookAPI(pydantic bool) string {
	return generateAPIPackage("catalog", pydantic)["views.py"]
}
