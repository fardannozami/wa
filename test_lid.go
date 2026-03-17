package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test_lid.go <tenant_id>")
		return
	}

	tenantID := os.Args[1]
	sessionDir := "./sessions"
	dbPath := filepath.Join(sessionDir, "wa_"+tenantID+".db") + "?_foreign_keys=on"

	dbLog := waLog.Stdout("Database", "ERROR", true)
	container, err := sqlstore.New(context.Background(), "sqlite3", dbPath, dbLog)
	if err != nil {
		fmt.Printf("Failed to create store: %v\n", err)
		return
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		fmt.Printf("Failed to get device: %v\n", err)
		return
	}

	contacts, err := deviceStore.Contacts.GetAllContacts(context.Background())
	if err != nil {
		fmt.Printf("Failed to get contacts: %v\n", err)
		return
	}

	fmt.Printf("Found %d contacts in store\n", len(contacts))
	count := 0
	for jid, contact := range contacts {
		if count < 10 {
			fmt.Printf("Contact: JID=%s, User=%s, Server=%s, FullName=%s, PushName=%s\n", jid.String(), jid.User, jid.Server, contact.FullName, contact.PushName)
		}
		count++
	}
}
