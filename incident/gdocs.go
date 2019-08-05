package incident

import (
	"blabber/bot"
	"context"
	"fmt"

	"google.golang.org/api/drive/v3"
	log "gopkg.in/inconshreveable/log15.v2"
)

// RemoteDocument is a simple interface for
// Interacting with different type of documents
type RemoteDocument interface {
	// Gets the remote document from a template
	NewFromTemplate(title string, c *bot.Configuration) error
	// Gets the remote document from its ID
	GetFromId(documentID string) error
	// Returns the url at which you can fetch the document.
	Url() string
	// Returns the unique ID that can be used for GetFromId later
	Id() string
}

// GoogleDoc is a RemoteDocument implementation that uses Google Docs.
type GoogleDoc struct {
	Config  *GoogleDriveConfig
	Service *drive.Service
	file    *drive.File
}

func foo(d RemoteDocument) {
	fmt.Printf(d.Url())
}

// NewGoogleDoc creates a new GoogleDoc instance.
// Takes the configuration as a parameter.
func NewGoogleDoc() *GoogleDoc {
	client, err := GDriveConfig.GetClient()
	if err != nil {
		log.Error("Could not initiate the connection to the google drive API", "error", err)
		return nil
	}
	service, err := drive.New(client)
	if err != nil {
		log.Error("Could not connect to the drive service", "error", err)
		return nil
	}
	doc := &GoogleDoc{Config: GDriveConfig, Service: service}
	return doc
}

// NewFromTemplate creates a new file copying the master
func (doc *GoogleDoc) NewFromTemplate(title string, c *bot.Configuration) error {
	templateID := c.DocTemplate
	parents := []string{c.DocFolder}
	driveID := c.DocDrive
	files := drive.NewFilesService(doc.Service)
	// Todo: add DriveId to the newly created file, to create it in a team drive, and then
	gFile, err := files.Copy(templateID, &drive.File{Name: title, Parents: parents, TeamDriveId: driveID}).SupportsTeamDrives(true).Context(context.TODO()).Do()
	if err != nil {
		return fmt.Errorf("Could not copy the template to a new file: %v", err)
	}
	// Set the correct permissions on the new file.
	permissions := drive.NewPermissionsService(doc.Service)
	if _, err = permissions.Create(gFile.Id, &drive.Permission{Domain: "wikimedia.org", AllowFileDiscovery: true, Type: "domain", Role: "writer"}).SupportsTeamDrives(true).Context(context.TODO()).Do(); err != nil {
		return fmt.Errorf("Error adding permissions to the document: %v", err)
	}
	doc.file = gFile
	return nil
}

// GetFromId  fetches the file by ID.
func (doc *GoogleDoc) GetFromId(documentID string) error {
	files := drive.NewFilesService(doc.Service)
	gfile, err := files.Get(documentID).Context(context.Background()).Do()
	if err != nil {
		return fmt.Errorf("Could not find the document with id %s: %v", documentID, err)
	}
	doc.file = gfile
	return nil
}

// Url returns the url you can reach your document at.
func (doc *GoogleDoc) Url() string {
	if doc.file == nil || doc.file.Id == "" {
		return "<not available>"
	}
	return fmt.Sprintf("https://docs.google.com/document/d/%s/edit", doc.file.Id)
}

// Id returns the file id, a string.
func (doc *GoogleDoc) Id() string {
	return doc.file.Id
}
