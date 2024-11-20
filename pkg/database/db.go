package database

type DB struct {
	UserStore     UserStore
	EventStore    EventStore
	EventLogStore EventLogStore
}
