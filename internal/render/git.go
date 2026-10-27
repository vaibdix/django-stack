package render

func (p Project) gitignoreFiles() FileSet {
	files := FileSet{}
	files.add(GitignoreFile, RenderStatic("gitignore.tmpl"), FileMode)
	return files
}
