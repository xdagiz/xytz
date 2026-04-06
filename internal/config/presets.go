package config

type QualityPreset struct {
	Name        string
	Description string
	Format      string
}

var QualityPresets = []QualityPreset{
	{
		Name:   "best",
		Format: "bv*+ba/b",
	},
	{
		Name:   "4k",
		Format: "bv[height<=2160]+ba/b[height<=2160]",
	},
	{
		Name:   "2k",
		Format: "bv[height<=1440]+ba/b[height<=1440]",
	},
	{
		Name:   "1080p",
		Format: "bv[height<=1080]+ba/b[height<=1080]",
	},
	{
		Name:   "720p",
		Format: "bv[height<=720]+ba/b[height<=720]",
	},
	{
		Name:   "480p",
		Format: "bv[height<=480]+ba/b[height<=480]",
	},
	{
		Name:   "360p",
		Format: "bv[height<=360]+ba/b[height<=360]",
	},
}

func PresetNames() []string {
	names := make([]string, len(QualityPresets))
	for i, p := range QualityPresets {
		names[i] = p.Name
	}

	return names
}

func GetPresetByName(name string) *QualityPreset {
	for i := range QualityPresets {
		if QualityPresets[i].Name == name {
			return &QualityPresets[i]
		}
	}

	return nil
}

func IsValidPreset(name string) bool {
	return GetPresetByName(name) != nil
}

func ResolveQuality(quality string) string {
	if quality == "" {
		return QualityPresets[0].Format
	}

	preset := GetPresetByName(quality)
	if preset != nil {
		return preset.Format
	}

	return quality
}
