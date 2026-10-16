package scaffold

func generateBookAPI() string {
	return generateAPIPackage("catalog")["views.py"]
}
