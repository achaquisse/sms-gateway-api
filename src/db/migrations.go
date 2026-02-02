package db

func RunMigrations() error {
	return DB.AutoMigrate(
		&Device{},
		&Message{},
		&DeviceTopic{},
		&SchemaMigration{},
	)
}

func InitSchema() error {
	return RunMigrations()
}

func GetCurrentVersion() (int, error) {
	var migration SchemaMigration
	err := DB.Order("version DESC").First(&migration).Error
	if err != nil {
		return 0, nil
	}
	return migration.Version, nil
}
