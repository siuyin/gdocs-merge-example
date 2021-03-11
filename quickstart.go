package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
)

// Retrieves a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Requests a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		log.Fatalf("Unable to cache OAuth token: %v", err)
	}
	json.NewEncoder(f).Encode(token)
}

// Experiments with permissions
// https://docs.google.com/document/d/195j9eDD3ccgjQRttHhJPymLJUCOUjs-jmwTrekvdjFE/edit
//docID := "195j9eDD3ccgjQRttHhJPymLJUCOUjs-jmwTrekvdjFE" // sample from google. OK
//docID := "1FIuZ4E7xNuSPbFK6fTh3fMNiJzsUZhdAWoK83_TbjrM" // my-drive instructional videos. OK
//docID := "1GyU79CvK-cObtGgMDEhMd8HHrQ__AYVUfkHaOMgSDhM" // firewarden shared file. No access
//docID := "15hzucVTwaX7fBrQcssHlUmrSdN3NarR1yxcHK9DZ-Us" // firewardens postmortems 101. No access
//docID := "1WUOdmYfqydN-U9RvvCS0BZ-j9TCGFdbeuCe0ZJgQ9Ag" // shared drive / MACS team drive. OK
//docID := "10ezfD5y1kXZyuyPrzfoqwnirFOelZEi0HrK94pYsPPI" // shared drive / postmortem file. OK
const (
	testDocID      = "1F6ye209lFqkg5LHCepvK2vQMDnCuxW_PagasjwWuq5o" // my-drive / try-gdocs-api. OK
	outputFilename = "merged-output-from-try-gdocs-api"
)

func main() {
	b := readCredentials("credentials.json")

	ds := newGoogleDriveService(b)
	outFile := copyGoogleDriveFile(ds, testDocID, outputFilename)

	docID := outFile.Id

	srv := newGoogleDocsService(b)
	//printDoc(srv, docID)
	updateDoc(srv, docID)
}

func copyGoogleDriveFile(ds *drive.Service, srcID, tgtName string) *drive.File {
	f, err := ds.Files.Copy(srcID, &drive.File{Name: tgtName}).Do()
	if err != nil {
		log.Fatalf("could not copy file: %v", err)
	}
	return f
}

func readCredentials(fn string) []byte {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	return b
}
func newGoogleDriveService(b []byte) *drive.Service {
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}
	return srv

}

func newGoogleDocsService(b []byte) *docs.Service {
	// If modifying these scopes, delete your previously saved token.json.
	//config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/documents.readonly") // see: https://developers.google.com/identity/protocols/oauth2/scopes#docs
	//config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/drive.readonly")
	//config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/drive.file") // Per-file access to files created or opened by the app. File authorization is granted on a per-user basis and is revoked when the user deauthorizes the app.
	//config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/documents") // View and manage your Google Docs documents
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := docs.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Docs client: %v", err)
	}
	return srv
}

func printDoc(srv *docs.Service, docID string) {
	doc := getDoc(srv, docID)
	fmt.Printf("The title of the doc is: %s\n", doc.Title)

	for i, v := range doc.Body.Content {
		b := getBodyContentJSON(v)
		showTables(v, i, b)
		showParagraphs(v, i, b)

	}
}

func getDoc(srv *docs.Service, docID string) *docs.Document {
	doc, err := srv.Documents.Get(docID).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from document: %v", err)
	}
	return doc
}

func getBodyContentJSON(se *docs.StructuralElement) []byte {
	b, err := se.MarshalJSON()
	if err != nil {
		log.Fatalf("getBodyContentJSON: %v", err)
	}
	return b
}

func showTables(se *docs.StructuralElement, i int, b []byte) {
	if se.Table == nil {
		return
	}
	fmt.Println("Table start index:", se.StartIndex)
	fmt.Printf("Element: %d\n\n-------\n", i)
}

func showParagraphs(se *docs.StructuralElement, i int, b []byte) {
	if se.Paragraph == nil {
		return
	}
	for _, e := range se.Paragraph.Elements {
		fmt.Println(e.TextRun.Content)
	}
	fmt.Printf("Element: %d\n\n%s\n-------\n", i, b)
}

const (
	insertIndex = 178
	tableIndex  = 82
)

func updateDoc(srv *docs.Service, docID string) {
	reqs := updateRequests()
	breq := docs.BatchUpdateDocumentRequest{Requests: reqs}
	bres, err := srv.Documents.BatchUpdate(docID, &breq).Do()
	if err != nil {
		log.Fatalf("Unabled to update data in document: %v", err)
	}
	showUpdateResponses(bres)
}

func updateRequests() []*docs.Request {
	// make change requests from the end of the doc towards the beginning.
	reqs := []*docs.Request{
		&docs.Request{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{Index: insertIndex},
				Text:     "\nHello.",
			},
		},
		&docs.Request{
			InsertTableRow: &docs.InsertTableRowRequest{
				InsertBelow:       true,
				TableCellLocation: &docs.TableCellLocation{ColumnIndex: 0, RowIndex: 0, TableStartLocation: &docs.Location{Index: tableIndex}},
			},
		},
		&docs.Request{
			ReplaceAllText: &docs.ReplaceAllTextRequest{
				ContainsText: &docs.SubstringMatchCriteria{Text: "{{insert2}}"},
				ReplaceText:  "lazy dog",
			},
		},
		&docs.Request{
			ReplaceAllText: &docs.ReplaceAllTextRequest{
				ContainsText: &docs.SubstringMatchCriteria{Text: "{{insert1}}"},
				ReplaceText:  "The Quick Brown fox.",
			},
		},
	}
	return reqs
}

func showUpdateResponses(res *docs.BatchUpdateDocumentResponse) {
	b, err := res.MarshalJSON()
	if err != nil {
		log.Fatalf("showUpdateResponses: %v", err)
	}
	fmt.Printf("%s\n", b)
}
