package database

import (
	"database/sql"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

type Node struct {
	IP          string    `json:"ip"`
	Country     string    `json:"country"`
	City        string    `json:"city"`
	Lat         float64   `json:"lat"`
	Lon         float64   `json:"lon"`
	ISP         string    `json:"isp"`
	LastSeen    time.Time `json:"lastSeen"`
	Region      string    `json:"region"`
	RegionName  string    `json:"regionName"`
	Timezone    string    `json:"timezone"`
	Zip         string    `json:"zip"`
	AS          string    `json:"as"`
	Org         string    `json:"org"`
	Query       string    `json:"query"`
	Status      string    `json:"status"`
	CountryCode string    `json:"countryCode"`
	District    string    `json:"district"`
	Continent   string    `json:"continent"`
	Currency    string    `json:"currency"`
	Mobile      bool      `json:"mobile"`
	Proxy       bool      `json:"proxy"`
	Hosting     bool      `json:"hosting"`
	IsOnline    bool      `json:"isOnline"`
	LastPing    time.Time `json:"lastPing"`
	FirstSeen   time.Time `json:"firstSeen"`
	TotalPings  int       `json:"totalPings"`
	OnlinePings int       `json:"onlinePings"`
	Uptime      int64     `json:"uptime"` // Uptime in seconds
	IsStaking   bool      `json:"isStaking"`
}

type DB struct {
	db *sql.DB
}

func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create nodes table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS nodes (
			ip TEXT PRIMARY KEY,
			country TEXT,
			city TEXT,
			lat REAL,
			lon REAL,
			isp TEXT,
			last_seen TIMESTAMP,
			region TEXT,
			region_name TEXT,
			timezone TEXT,
			zip TEXT,
			as_number TEXT,
			org TEXT,
			query TEXT,
			status TEXT,
			country_code TEXT,
			district TEXT,
			continent TEXT,
			currency TEXT,
			mobile BOOLEAN,
			proxy BOOLEAN,
			hosting BOOLEAN,
			is_online BOOLEAN,
			last_ping TIMESTAMP,
			first_seen TIMESTAMP,
			total_pings INTEGER DEFAULT 0,
			online_pings INTEGER DEFAULT 0,
			uptime INTEGER DEFAULT 0,
			is_staking BOOLEAN DEFAULT FALSE
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create peers table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS peers (
			ip TEXT PRIMARY KEY
		)
	`)
	if err != nil {
		return nil, err
	}

	// Add is_staking column if it doesn't exist
	_, err = db.Exec(`
		ALTER TABLE nodes ADD COLUMN is_staking BOOLEAN DEFAULT FALSE;
	`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) UpsertNode(node *Node) error {
	log.Debug().
		Str("ip", node.IP).
		Str("country", node.Country).
		Str("city", node.City).
		Float64("lat", node.Lat).
		Float64("lon", node.Lon).
		Msg("Upserting node to database")

	_, err := d.db.Exec(`
		INSERT INTO nodes (
			ip, country, city, lat, lon, isp, last_seen,
			region, region_name, timezone, zip, as_number, org, query, status,
			country_code, district, continent, currency, mobile, proxy, hosting,
			is_online, last_ping, first_seen, total_pings, online_pings, uptime,
			is_staking
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(ip) DO UPDATE SET
			country = excluded.country,
			city = excluded.city,
			lat = excluded.lat,
			lon = excluded.lon,
			isp = excluded.isp,
			last_seen = excluded.last_seen,
			region = excluded.region,
			region_name = excluded.region_name,
			timezone = excluded.timezone,
			zip = excluded.zip,
			as_number = excluded.as_number,
			org = excluded.org,
			query = excluded.query,
			status = excluded.status,
			country_code = excluded.country_code,
			district = excluded.district,
			continent = excluded.continent,
			currency = excluded.currency,
			mobile = excluded.mobile,
			proxy = excluded.proxy,
			hosting = excluded.hosting,
			is_online = excluded.is_online,
			last_ping = excluded.last_ping,
			first_seen = excluded.first_seen,
			total_pings = excluded.total_pings,
			online_pings = excluded.online_pings,
			uptime = excluded.uptime,
			is_staking = excluded.is_staking
	`, node.IP, node.Country, node.City, node.Lat, node.Lon, node.ISP, node.LastSeen,
		node.Region, node.RegionName, node.Timezone, node.Zip, node.AS, node.Org, node.Query, node.Status,
		node.CountryCode, node.District, node.Continent, node.Currency, node.Mobile, node.Proxy, node.Hosting,
		node.IsOnline, node.LastPing, node.FirstSeen, node.TotalPings, node.OnlinePings, node.Uptime, node.IsStaking)

	if err != nil {
		log.Error().Err(err).Str("ip", node.IP).Msg("Error upserting node")
		return err
	}

	log.Debug().Str("ip", node.IP).Msg("Successfully upserted node")
	return nil
}

func (d *DB) GetNode(ip string) (*Node, error) {
	var node Node
	err := d.db.QueryRow(`
		SELECT ip, country, city, lat, lon, isp, last_seen,
			region, region_name, timezone, zip, as_number, org, query, status,
			country_code, district, continent, currency, mobile, proxy, hosting,
			is_online, last_ping, first_seen, total_pings, online_pings, uptime,
			is_staking
		FROM nodes
		WHERE ip = ?
	`, ip).Scan(
		&node.IP, &node.Country, &node.City, &node.Lat, &node.Lon, &node.ISP, &node.LastSeen,
		&node.Region, &node.RegionName, &node.Timezone, &node.Zip, &node.AS, &node.Org, &node.Query, &node.Status,
		&node.CountryCode, &node.District, &node.Continent, &node.Currency, &node.Mobile, &node.Proxy, &node.Hosting,
		&node.IsOnline, &node.LastPing, &node.FirstSeen, &node.TotalPings, &node.OnlinePings, &node.Uptime, &node.IsStaking)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (d *DB) GetAllNodes() ([]*Node, error) {
	log.Debug().Msg("Retrieving all nodes from database")

	rows, err := d.db.Query(`
		SELECT ip, country, city, lat, lon, isp, last_seen,
			region, region_name, timezone, zip, as_number, org, query, status,
			country_code, district, continent, currency, mobile, proxy, hosting,
			is_online, last_ping, first_seen, total_pings, online_pings, uptime,
			is_staking
		FROM nodes
		ORDER BY last_seen DESC
	`)
	if err != nil {
		log.Error().Err(err).Msg("Error querying nodes")
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		var node Node
		err := rows.Scan(
			&node.IP, &node.Country, &node.City, &node.Lat, &node.Lon, &node.ISP, &node.LastSeen,
			&node.Region, &node.RegionName, &node.Timezone, &node.Zip, &node.AS, &node.Org, &node.Query, &node.Status,
			&node.CountryCode, &node.District, &node.Continent, &node.Currency, &node.Mobile, &node.Proxy, &node.Hosting,
			&node.IsOnline, &node.LastPing, &node.FirstSeen, &node.TotalPings, &node.OnlinePings, &node.Uptime, &node.IsStaking)
		if err != nil {
			log.Error().Err(err).Msg("Error scanning node row")
			return nil, err
		}
		nodes = append(nodes, &node)
	}

	log.Debug().Int("count", len(nodes)).Msg("Retrieved nodes from database")
	return nodes, rows.Err()
}

func (d *DB) GetStaleNodes(olderThan time.Duration) ([]string, error) {
	rows, err := d.db.Query(`
		SELECT ip
		FROM nodes
		WHERE last_seen < datetime('now', ?)
	`, olderThan.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		ips = append(ips, ip)
	}
	return ips, rows.Err()
}

func (d *DB) UpdateNodeStatus(ip string, isOnline bool) error {
	now := time.Now()
	log.Debug().
		Str("ip", ip).
		Bool("isOnline", isOnline).
		Time("now", now).
		Msg("Updating node status")

	// Get current node stats
	var node Node
	err := d.db.QueryRow("SELECT first_seen, total_pings, online_pings, uptime, is_online, last_ping FROM nodes WHERE ip = ?", ip).Scan(
		&node.FirstSeen,
		&node.TotalPings,
		&node.OnlinePings,
		&node.Uptime,
		&node.IsOnline,
		&node.LastPing,
	)
	if err != nil && err != sql.ErrNoRows {
		log.Error().Err(err).Str("ip", ip).Msg("Error getting node stats")
		return err
	}

	// If this is the first ping, set FirstSeen
	if node.FirstSeen.IsZero() {
		node.FirstSeen = now
		log.Debug().Str("ip", ip).Time("firstSeen", now).Msg("Setting first seen time")
	}

	// Update ping statistics
	node.TotalPings++
	if isOnline {
		node.OnlinePings++
		// Update uptime to be the time since last successful ping
		if !node.LastPing.IsZero() {
			node.Uptime = int64(now.Sub(node.LastPing).Seconds())
		}
	} else {
		// If node is offline, uptime is 0
		node.Uptime = 0
	}

	// Check if node is staking (online for more than an hour with high uptime)
	isStaking := false
	if node.TotalPings >= 12 && // At least 12 pings (2 minutes * 12 = 24 minutes minimum)
		time.Since(node.FirstSeen) >= time.Hour && // Running for at least an hour
		float64(node.OnlinePings)/float64(node.TotalPings) >= 0.95 { // 95% successful pings
		isStaking = true
		log.Debug().
			Str("ip", ip).
			Int64("uptime", node.Uptime).
			Int("totalPings", node.TotalPings).
			Int("onlinePings", node.OnlinePings).
			Msg("Node marked as staking")
	}

	// Update the node
	_, err = d.db.Exec(`
		UPDATE nodes 
		SET is_online = ?, 
			last_ping = ?, 
			total_pings = ?,
			online_pings = ?,
			uptime = ?,
			is_staking = ?
		WHERE ip = ?
	`, isOnline, now, node.TotalPings, node.OnlinePings, node.Uptime, isStaking, ip)

	if err != nil {
		log.Error().
			Err(err).
			Str("ip", ip).
			Bool("isOnline", isOnline).
			Int64("uptime", node.Uptime).
			Msg("Error updating node status")
		return err
	}

	log.Debug().
		Str("ip", ip).
		Bool("isOnline", isOnline).
		Int64("uptime", node.Uptime).
		Int("totalPings", node.TotalPings).
		Int("onlinePings", node.OnlinePings).
		Msg("Updated node status")

	return nil
}

// Add new function to save peers
func (d *DB) SavePeers(peers []string) error {
	// Start a transaction
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing peers
	_, err = tx.Exec("DELETE FROM peers")
	if err != nil {
		return err
	}

	// Insert new peers
	stmt, err := tx.Prepare("INSERT INTO peers (ip) VALUES (?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, peer := range peers {
		_, err = stmt.Exec(peer)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Add new function to load peers
func (d *DB) LoadPeers() ([]string, error) {
	rows, err := d.db.Query("SELECT ip FROM peers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, err
		}
		peers = append(peers, ip)
	}
	return peers, rows.Err()
}
