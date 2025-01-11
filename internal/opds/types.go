package opds

import (
	"encoding/xml"
	"time"
)

// Relations

const (
	RelSelf        string = "self"
	RelStart       string = "start"
	RelSearch      string = "search"
	RelSubsection  string = "subsection"
	RelCover       string = "http://opds-spec.org/image"
	RelThumbnail   string = "http://opds-spec.org/image/thumbnail"
	RelAcquisition string = "http://opds-spec.org/acquisition"
)

// Feed Types

const (
	FeedTypeNavigation     string = "application/atom+xml;profile=opds-catalog;kind=navigation"
	FeedTypeAcquisition    string = "application/atom+xml;profile=opds-catalog;kind=acquisition"
	FeedTypeSearchTemplate string = "application/atom+xml"
)

const (
	FileTypeCBZ string = "application/x-cbz"
)

type Time time.Time

func (t *Time) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement((*time.Time)(t).UTC().Format(time.RFC3339), start)
}

func TimeNow() Time {
	return Time(time.Now())
}

type Author struct {
	XMLName xml.Name `xml:"author"`
	Name    string   `xml:"name"`
	URI     string   `xml:"uri"`
}

type Link struct {
	XMLName xml.Name `xml:"link"`
	Href    string   `xml:"href,attr"`
	Rel     string   `xml:"rel,attr,omitempty"`
	Type    string   `xml:"type,attr,omitempty"`
	Title   string   `xml:"title,attr,omitempty"`
}

// https://specs.opds.io/opds-1.2#5-opds-catalog-entry-documents
type Entry struct {
	Title       string `xml:"title"`
	LastUpdated Time   `xml:"updated"`
	ID          string `xml:"id"`
	Content     string `xml:"content"` // Empty but tag should still exist
	Link        []Link `xml:"link"`
}

// https://specs.opds.io/opds-1.2#2-opds-catalog-feed-documents
type Feed struct {
	XMLName xml.Name `xml:"feed"`

	ID          string  `xml:"id"`      // required
	Title       string  `xml:"title"`   // required
	LastUpdated Time    `xml:"updated"` // required
	Author      *Author `xml:"author,omitempty"`
	Links       []Link  `xml:"link"`

	Entries []Entry `xml:"entry"`
}

type feedWithMetadata struct {
	*Feed

	Xmlns        string `xml:"xmlns,attr"`
	ThrXmlns     string `xml:"xmlns:thr,attr"`
	DctermsXmlns string `xml:"xmlns:dcterms,attr"`
	OpdsXmlns    string `xml:"xmlns:opds,attr"`
}

func (f *Feed) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	ff := feedWithMetadata{
		Feed:         f,
		ThrXmlns:     "http://purl.org/syndication/thread/1.0",
		DctermsXmlns: "http://purl.org/dc/terms/",
		OpdsXmlns:    "http://opds-spec.org/2010/catalog",
		Xmlns:        "http://www.w3.org/2005/Atom",
	}

	return e.EncodeElement(ff, start)
}
