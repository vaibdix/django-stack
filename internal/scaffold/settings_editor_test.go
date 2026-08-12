package scaffold

func (s *SettingsEditor) Content() string {
	return s.content
}

func (s *SettingsEditor) SetContent(content string) {
	s.content = content
}
