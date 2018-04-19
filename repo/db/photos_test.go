package db

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/textileio/textile-go/repo"
	"github.com/textileio/textile-go/repo/photos"
)

var phdb repo.PhotoStore

func init() {
	setupDB()
}

func setupDB() {
	conn, _ := sql.Open("sqlite3", ":memory:")
	initDatabaseTables(conn, "")
	phdb = NewPhotoStore(conn, new(sync.Mutex))
}

func TestPhotoDB_Put(t *testing.T) {
	md := &photos.Metadata{}
	err := phdb.Put("Qmabc123", "", md)
	if err != nil {
		t.Error(err)
	}
	stmt, err := phdb.PrepareQuery("select cid from photos where cid=?")
	defer stmt.Close()
	var cid string
	err = stmt.QueryRow("Qmabc123").Scan(&cid)
	if err != nil {
		t.Error(err)
	}
	if cid != "Qmabc123" {
		t.Errorf(`expected "Qmabc123" got %s`, cid)
	}
}

func TestPhotoDB_GetPhotos(t *testing.T) {
	setupDB()
	md := &photos.Metadata{
		Added: time.Now(),
	}
	err := phdb.Put("Qm123", "", md)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second * 1)
	md2 := &photos.Metadata{
		Added: time.Now(),
	}
	err = phdb.Put("Qm456", "Qm123", md2)
	if err != nil {
		t.Error(err)
	}
	ps := phdb.GetPhotos("", -1)
	if len(ps) != 2 {
		t.Error("returned incorrect number of photos")
		return
	}

	limited := phdb.GetPhotos("", 1)
	if len(limited) != 1 {
		t.Error("returned incorrect number of photos")
		return
	}

	offset := phdb.GetPhotos(limited[0].Cid, -1)
	if len(offset) != 1 {
		t.Error("returned incorrect number of photos")
		return
	}
}

func TestPhotoDB_DeletePhoto(t *testing.T) {
	setupDB()
	md := &photos.Metadata{}
	err := phdb.Put("Qm789", "", md)
	if err != nil {
		t.Error(err)
	}
	ps := phdb.GetPhotos("", -1)
	if len(ps) == 0 {
		t.Error("Returned incorrect number of photos")
		return
	}
	err = phdb.DeletePhoto(ps[0].Cid)
	if err != nil {
		t.Error(err)
	}
	stmt, err := phdb.PrepareQuery("select cid from photos where cid=?")
	defer stmt.Close()
	var cid string
	err = stmt.QueryRow(ps[0].Cid).Scan(&cid)
	if err == nil {
		t.Error("Delete failed")
	}
}
