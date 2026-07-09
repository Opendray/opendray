package dbtool

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrReadOnlyConnection is returned when a write reaches a connection
// flagged read_only.
var ErrReadOnlyConnection = errors.New("dbtool: connection is read-only")

// ErrWriteScope is returned when a non-read statement arrives without
// write authorization.
var ErrWriteScope = errors.New("dbtool: statement modifies data — requires the db:write scope")

// ErrNoPrimaryKey is returned when row editing is attempted on a table
// without a primary key.
var ErrNoPrimaryKey = errors.New("dbtool: table has no primary key — row editing is disabled")

// ErrUnsupportedDriver is returned for a driver name we don't ship.
var ErrUnsupportedDriver = errors.New("dbtool: unsupported driver (supported: postgres, mysql, mariadb, sqlite)")

// Options are the service's runtime knobs, resolved from [dbtool] config.
type Options struct {
	QueryTimeout time.Duration // per-statement cap; 0 → 30s
	MaxRows      int           // default row cap; 0 → 500 (hard cap 10000)
	PoolMaxConns int           // per-connection pgx pool cap; 0 → 3
	PoolIdleTTL  time.Duration // idle pool eviction; 0 → 5m
}

const hardMaxRows = 10000

func (o Options) queryTimeout() time.Duration {
	if o.QueryTimeout > 0 {
		return o.QueryTimeout
	}
	return 30 * time.Second
}

func (o Options) maxRows() int {
	if o.MaxRows > 0 {
		return o.MaxRows
	}
	return 500
}

func (o Options) poolIdleTTL() time.Duration {
	if o.PoolIdleTTL > 0 {
		return o.PoolIdleTTL
	}
	return 5 * time.Minute
}

type poolEntry struct {
	handle   Handle
	lastUsed time.Time
}

// Service owns the connection registry and a cache of open handles to
// the target databases (one per connection id, closed on idle-eviction,
// update, delete and shutdown).
type Service struct {
	store   *Store
	opts    Options
	log     *slog.Logger
	drivers map[string]Driver

	mu    sync.Mutex
	pools map[string]*poolEntry

	stop     chan struct{}
	stopOnce sync.Once
}

// NewService builds the service and starts the idle-eviction loop.
func NewService(store *Store, opts Options, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	s := &Service{
		store: store,
		opts:  opts,
		log:   log.With("component", "dbtool"),
		drivers: map[string]Driver{
			"postgres": postgresDriver{},
			"mysql":    mysqlDriver{},
			"mariadb":  mysqlDriver{}, // same wire protocol as MySQL
			"sqlite":   sqliteDriver{},
		},
		pools: map[string]*poolEntry{},
		stop:  make(chan struct{}),
	}
	go s.evictLoop()
	return s
}

// Close shuts down every cached handle. Idempotent.
func (s *Service) Close() {
	s.stopOnce.Do(func() { close(s.stop) })
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, e := range s.pools {
		e.handle.Close()
		delete(s.pools, id)
	}
}

func (s *Service) evictLoop() {
	ttl := s.opts.poolIdleTTL()
	t := time.NewTicker(ttl / 2)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			cutoff := time.Now().Add(-ttl)
			s.mu.Lock()
			for id, e := range s.pools {
				if e.lastUsed.Before(cutoff) {
					e.handle.Close()
					delete(s.pools, id)
					s.log.Debug("evicted idle pool", "connection", id)
				}
			}
			s.mu.Unlock()
		}
	}
}

// invalidate closes and forgets the cached handle for id.
func (s *Service) invalidate(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.pools[id]; ok {
		e.handle.Close()
		delete(s.pools, id)
	}
}

// handle returns the cached (or freshly opened) handle for conn.
func (s *Service) handle(ctx context.Context, conn Connection) (Handle, error) {
	s.mu.Lock()
	if e, ok := s.pools[conn.ID]; ok {
		e.lastUsed = time.Now()
		s.mu.Unlock()
		return e.handle, nil
	}
	s.mu.Unlock()

	drv, err := s.driver(conn.Driver)
	if err != nil {
		return nil, err
	}
	h, err := drv.Open(ctx, conn, s.driverOpts())
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.pools[conn.ID]; ok {
		// Lost the race — keep the first handle.
		h.Close()
		e.lastUsed = time.Now()
		return e.handle, nil
	}
	s.pools[conn.ID] = &poolEntry{handle: h, lastUsed: time.Now()}
	return h, nil
}

func (s *Service) driver(name string) (Driver, error) {
	d, ok := s.drivers[name]
	if !ok {
		return nil, ErrUnsupportedDriver
	}
	return d, nil
}

func (s *Service) driverOpts() DriverOpts {
	maxConns := s.opts.PoolMaxConns
	if maxConns <= 0 {
		maxConns = 3
	}
	return DriverOpts{MaxConns: maxConns, ConnectTimeout: 5 * time.Second}
}

// effectiveMaxRows resolves a request's row cap against config defaults
// and the hard ceiling.
func (s *Service) effectiveMaxRows(requested int) int {
	if requested <= 0 {
		return s.opts.maxRows()
	}
	if requested > hardMaxRows {
		return hardMaxRows
	}
	return requested
}

// ---- connection registry ----

// CreateParams is the payload for registering a connection.
type CreateParams struct {
	Cwd      string         `json:"cwd"`
	Name     string         `json:"name"`
	Driver   string         `json:"driver"`
	Host     string         `json:"host"`
	Port     int            `json:"port"`
	DBName   string         `json:"db_name"`
	Username string         `json:"username"`
	Password string         `json:"password"`
	SSLMode  string         `json:"ssl_mode"`
	ReadOnly bool           `json:"read_only"`
	Options  map[string]any `json:"options"`
}

func (p CreateParams) validate() error {
	if strings.TrimSpace(p.Cwd) == "" {
		return errors.New("dbtool: cwd is required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("dbtool: name is required")
	}
	// SQLite is a file-path connection: db_name holds the path (validated
	// against the project cwd by the driver at Open time); there is no
	// host / port / username.
	if p.Driver == "sqlite" {
		if strings.TrimSpace(p.DBName) == "" {
			return errors.New("dbtool: sqlite requires a database file path")
		}
		return nil
	}
	if strings.TrimSpace(p.Host) == "" {
		return errors.New("dbtool: host is required")
	}
	if strings.TrimSpace(p.DBName) == "" {
		return errors.New("dbtool: db_name is required")
	}
	if strings.TrimSpace(p.Username) == "" {
		return errors.New("dbtool: username is required")
	}
	if p.Port < 0 || p.Port > 65535 {
		return fmt.Errorf("dbtool: invalid port %d", p.Port)
	}
	return nil
}

// normalize fills defaults into p.
func (p *CreateParams) normalize() {
	if p.Driver == "" {
		p.Driver = "postgres"
	}
	if p.Port == 0 {
		p.Port = defaultPort(p.Driver)
	}
	if p.SSLMode == "" {
		p.SSLMode = "prefer"
	}
	if p.Options == nil {
		p.Options = map[string]any{}
	}
}

// defaultPort is the well-known port for a driver (0 for the file-based
// SQLite, which has no port).
func defaultPort(driver string) int {
	switch driver {
	case "mysql", "mariadb":
		return 3306
	case "sqlite":
		return 0
	default:
		return 5432
	}
}

// CreateConnection validates and stores a new connection.
func (s *Service) CreateConnection(ctx context.Context, p CreateParams) (Connection, error) {
	p.normalize()
	if err := p.validate(); err != nil {
		return Connection{}, err
	}
	if _, err := s.driver(p.Driver); err != nil {
		return Connection{}, err
	}
	now := time.Now().UTC()
	c := Connection{
		ID:        newID(),
		Cwd:       p.Cwd,
		Name:      p.Name,
		Driver:    p.Driver,
		Host:      p.Host,
		Port:      p.Port,
		DBName:    p.DBName,
		Username:  p.Username,
		Password:  p.Password,
		SSLMode:   p.SSLMode,
		ReadOnly:  p.ReadOnly,
		Options:   p.Options,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.Insert(ctx, c); err != nil {
		return Connection{}, err
	}
	c.HasPassword = c.Password != ""
	return c, nil
}

// ListConnections returns the registry for cwd (all projects when empty).
func (s *Service) ListConnections(ctx context.Context, cwd string) ([]Connection, error) {
	return s.store.List(ctx, cwd)
}

// GetConnection returns one connection.
func (s *Service) GetConnection(ctx context.Context, id string) (Connection, error) {
	return s.store.Get(ctx, id)
}

// UpdateConnection applies patch and invalidates the cached handle.
func (s *Service) UpdateConnection(ctx context.Context, id string, patch UpdatePatch) (Connection, error) {
	c, err := s.store.Update(ctx, id, patch)
	if err != nil {
		return Connection{}, err
	}
	s.invalidate(id)
	return c, nil
}

// DeleteConnection removes the connection and closes its handle.
func (s *Service) DeleteConnection(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidate(id)
	return nil
}

// TestParams tests connectivity for an unsaved connection payload.
func (s *Service) TestParams(ctx context.Context, p CreateParams) PingResult {
	p.normalize()
	if err := p.validate(); err != nil {
		return PingResult{OK: false, Error: err.Error()}
	}
	drv, err := s.driver(p.Driver)
	if err != nil {
		return PingResult{OK: false, Error: err.Error()}
	}
	c := Connection{
		Host: p.Host, Port: p.Port, DBName: p.DBName,
		Username: p.Username, Password: p.Password, SSLMode: p.SSLMode,
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return drv.Ping(ctx, c, s.driverOpts())
}

// TestConnection tests connectivity using the stored credentials.
func (s *Service) TestConnection(ctx context.Context, id string) (PingResult, error) {
	c, err := s.store.Get(ctx, id)
	if err != nil {
		return PingResult{}, err
	}
	drv, err := s.driver(c.Driver)
	if err != nil {
		return PingResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return drv.Ping(ctx, c, s.driverOpts()), nil
}

// ---- data access ----

// withConn resolves id to (driver, handle, connection).
func (s *Service) withConn(ctx context.Context, id string) (Driver, Handle, Connection, error) {
	c, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, nil, Connection{}, err
	}
	drv, err := s.driver(c.Driver)
	if err != nil {
		return nil, nil, Connection{}, err
	}
	h, err := s.handle(ctx, c)
	if err != nil {
		return nil, nil, Connection{}, err
	}
	return drv, h, c, nil
}

// Schemas lists the target database's schemas.
func (s *Service) Schemas(ctx context.Context, id string) ([]Schema, error) {
	drv, h, _, err := s.withConn(ctx, id)
	if err != nil {
		return nil, err
	}
	return drv.Schemas(ctx, h, s.opts.queryTimeout())
}

// Tables lists one schema's tables and views.
func (s *Service) Tables(ctx context.Context, id, schema string) ([]Table, error) {
	drv, h, _, err := s.withConn(ctx, id)
	if err != nil {
		return nil, err
	}
	return drv.Tables(ctx, h, schema, s.opts.queryTimeout())
}

// TableMeta describes one table.
func (s *Service) TableMeta(ctx context.Context, id, schema, table string) (TableMeta, error) {
	drv, h, _, err := s.withConn(ctx, id)
	if err != nil {
		return TableMeta{}, err
	}
	return drv.TableMeta(ctx, h, schema, table, s.opts.queryTimeout())
}

// TableData pages through one table (always via the read fence).
func (s *Service) TableData(ctx context.Context, id string, req TableDataReq) (*ResultSet, error) {
	drv, h, _, err := s.withConn(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Limit <= 0 || req.Limit > hardMaxRows {
		req.Limit = 100
	}
	return drv.TableData(ctx, h, req, s.opts.queryTimeout())
}

// checkEditable enforces the write preconditions for row CRUD: a
// writable connection and (for update/delete) a full primary key.
func (s *Service) checkEditable(ctx context.Context, drv Driver, h Handle, c Connection, schema, table string, pks []map[string]any) error {
	if c.ReadOnly {
		return ErrReadOnlyConnection
	}
	if pks == nil {
		return nil // insert path — no PK requirement
	}
	meta, err := drv.TableMeta(ctx, h, schema, table, s.opts.queryTimeout())
	if err != nil {
		return err
	}
	if len(meta.PrimaryKey) == 0 {
		return ErrNoPrimaryKey
	}
	want := append([]string(nil), meta.PrimaryKey...)
	sort.Strings(want)
	for _, pk := range pks {
		got := sortedKeys(pk)
		if len(got) != len(want) {
			return fmt.Errorf("dbtool: primary key must include exactly %v", meta.PrimaryKey)
		}
		for i := range got {
			if got[i] != want[i] {
				return fmt.Errorf("dbtool: primary key must include exactly %v", meta.PrimaryKey)
			}
		}
	}
	return nil
}

// InsertRow inserts one row and returns it.
func (s *Service) InsertRow(ctx context.Context, id string, req RowInsertReq) (*ResultSet, error) {
	drv, h, c, err := s.withConn(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.checkEditable(ctx, drv, h, c, req.Schema, req.Table, nil); err != nil {
		return nil, err
	}
	return drv.InsertRow(ctx, h, req, s.opts.queryTimeout())
}

// UpdateRow updates one row addressed by its full primary key.
func (s *Service) UpdateRow(ctx context.Context, id string, req RowUpdateReq) (int64, error) {
	drv, h, c, err := s.withConn(ctx, id)
	if err != nil {
		return 0, err
	}
	if err := s.checkEditable(ctx, drv, h, c, req.Schema, req.Table, []map[string]any{req.PK}); err != nil {
		return 0, err
	}
	return drv.UpdateRow(ctx, h, req, s.opts.queryTimeout())
}

// DeleteRows deletes rows addressed by their full primary keys.
func (s *Service) DeleteRows(ctx context.Context, id string, req RowDeleteReq) (int64, error) {
	drv, h, c, err := s.withConn(ctx, id)
	if err != nil {
		return 0, err
	}
	if err := s.checkEditable(ctx, drv, h, c, req.Schema, req.Table, req.PKs); err != nil {
		return 0, err
	}
	return drv.DeleteRows(ctx, h, req, s.opts.queryTimeout())
}

// Query classifies and runs one console statement. allowWrite reflects
// the caller's authorization (admin or db:write); the connection's
// read_only flag trumps it. Read statements always execute through the
// read-only-transaction fence regardless of authorization.
func (s *Service) Query(ctx context.Context, id string, req QueryReq, allowWrite bool) (*ResultSet, error) {
	class, err := Classify(req.SQL)
	if err != nil {
		return nil, err
	}
	drv, h, c, err := s.withConn(ctx, id)
	if err != nil {
		return nil, err
	}
	if class != ClassRead {
		if !allowWrite {
			return nil, ErrWriteScope
		}
		if c.ReadOnly {
			return nil, ErrReadOnlyConnection
		}
	}
	maxRows := s.effectiveMaxRows(req.MaxRows)
	return drv.Query(ctx, h, req, class, maxRows, s.opts.queryTimeout())
}

func newID() string {
	var b [9]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand should never fail; if it somehow does, refuse to
		// mint a weak (all-zero) id rather than silently continue.
		panic(fmt.Sprintf("dbtool: crypto/rand failed: %v", err))
	}
	return "dbc_" + base64.RawURLEncoding.EncodeToString(b[:])
}
