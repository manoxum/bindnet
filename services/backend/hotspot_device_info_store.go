package main

import "database/sql"

func hotspotDeviceInfoMap(db *sql.DB) (map[string]hotspotDeviceInfo, error) {
	rows, err := db.Query(`
		SELECT mac_address, vendor, device_name, os_name, confidence
		FROM hotspot_device_info
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	infos := map[string]hotspotDeviceInfo{}
	for rows.Next() {
		var info hotspotDeviceInfo
		if err := rows.Scan(&info.MACAddress, &info.Vendor, &info.DeviceName, &info.OSName, &info.Confidence); err != nil {
			return nil, err
		}
		infos[info.MACAddress] = info
	}
	return infos, rows.Err()
}

func hotspotDeviceInfoByMAC(db *sql.DB, mac string) (hotspotDeviceInfo, bool, error) {
	var info hotspotDeviceInfo
	err := db.QueryRow(`
		SELECT mac_address, vendor, device_name, os_name, confidence
		FROM hotspot_device_info
		WHERE mac_address = $1
	`, mac).Scan(&info.MACAddress, &info.Vendor, &info.DeviceName, &info.OSName, &info.Confidence)
	if err == nil {
		return info, hotspotDeviceInfoHasData(info), nil
	}
	if err == sql.ErrNoRows {
		return hotspotDeviceInfo{}, false, nil
	}
	return hotspotDeviceInfo{}, false, err
}

func hotspotDeviceInfoHasData(info hotspotDeviceInfo) bool {
	return (info.Vendor.Valid && info.Vendor.String != "") ||
		(info.DeviceName.Valid && info.DeviceName.String != "") ||
		(info.OSName.Valid && info.OSName.String != "") ||
		info.Confidence.Valid
}

func upsertHotspotDeviceInfo(db *sql.DB, info hotspotDeviceInfo) error {
	_, err := db.Exec(`
		INSERT INTO hotspot_device_info (mac_address, vendor, device_name, os_name, confidence, fetched_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT (mac_address) DO UPDATE
		SET vendor = EXCLUDED.vendor,
		    device_name = EXCLUDED.device_name,
		    os_name = EXCLUDED.os_name,
		    confidence = EXCLUDED.confidence,
		    fetched_at = CURRENT_TIMESTAMP
	`, info.MACAddress, nullableString(info.Vendor), nullableString(info.DeviceName), nullableString(info.OSName), nullableInt(info.Confidence))
	return err
}

func nullableString(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	return value.String
}

func nullableInt(value sql.NullInt64) any {
	if !value.Valid {
		return nil
	}
	return value.Int64
}
