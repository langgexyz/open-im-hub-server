package store

import "database/sql"

type DeviceToken struct {
	AppUID   string
	Platform int8
	Token    string
}

type DeviceTokenStore struct{ db *sql.DB }

func (s *DeviceTokenStore) Upsert(appUID string, platform int8, token string) error {
	_, err := s.db.Exec(
		`INSERT INTO device_tokens (uid, platform, token) VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE token = VALUES(token)`,
		appUID, platform, token,
	)
	return err
}

func (s *DeviceTokenStore) GetByUIDs(appUIDs []string) (map[string][]DeviceToken, error) {
	if len(appUIDs) == 0 {
		return nil, nil
	}
	args := make([]any, len(appUIDs))
	placeholders := ""
	for i, uid := range appUIDs {
		args[i] = uid
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
	}
	rows, err := s.db.Query(
		`SELECT uid, platform, token FROM device_tokens WHERE uid IN (`+placeholders+`)`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]DeviceToken)
	for rows.Next() {
		var dt DeviceToken
		if err := rows.Scan(&dt.AppUID, &dt.Platform, &dt.Token); err != nil {
			return nil, err
		}
		result[dt.AppUID] = append(result[dt.AppUID], dt)
	}
	return result, rows.Err()
}
