package crawlerbee

import (
	"github.com/muesli/beehive/bees"
)

// CrawlerBeeFactory is a factory for CrawlerBees.
type CrawlerBeeFactory struct {
	bees.BeeFactory
}

// New returns a new Bee instance configured with the supplied options.
func (factory *CrawlerBeeFactory) New(name, description string, options bees.BeeOptions) bees.BeeInterface {
	bee := CrawlerBee{
		Bee: bees.NewBee(name, factory.ID(), description, options),
	}
	bee.ReloadOptions(options)

	return &bee
}

// ID returns the ID of this Bee.
func (factory *CrawlerBeeFactory) ID() string {
	return "crawlerbee"
}

// Name returns the name of this Bee.
func (factory *CrawlerBeeFactory) Name() string {
	return "crawler"
}

// Description returns the description of this Bee.
func (factory *CrawlerBeeFactory) Description() string {
	return "Reacts to Web-feed updates"
}

// Image returns the filename of an image for this Bee.
func (factory *CrawlerBeeFactory) Image() string {
	return factory.ID() + ".png"
}

// LogoColor returns the preferred logo background color (used by the admin interface).
func (factory *CrawlerBeeFactory) LogoColor() string {
	return "#66b2bd"
}

// Options returns the options available to configure this Bee.
func (factory *CrawlerBeeFactory) Options() []bees.BeeOptionDescriptor {
	opts := []bees.BeeOptionDescriptor{
		{
			Name:        "url",
			Description: "URL of the web-feed you want to monitor",
			Type:        "url",
			Mandatory:   true,
		},
		{
			Name:        "feed_sel",
			Description: "Feed selector of the web-feed",
			Type:        "string",
			Mandatory:   true,
		},
		{
			Name:        "title_sel",
			Description: "title selector of the web-feed",
			Type:        "string",
			Mandatory:   true,
		},
		{
			Name:        "description_sel",
			Description: "description selector of the web-feed",
			Type:        "string",
			Mandatory:   false,
		},
		{
			Name:        "url_sel",
			Description: "url selector of the web-feed",
			Type:        "string",
			Mandatory:   false,
		},
		{
			Name:        "skip_first",
			Description: "Whether to skip already existing entries",
			Type:        "bool",
			Mandatory:   false,
		},
	}
	return opts
}

// Events describes the available events provided by this Bee.
func (factory *CrawlerBeeFactory) Events() []bees.EventDescriptor {
	events := []bees.EventDescriptor{
		{
			Namespace:   factory.Name(),
			Name:        "new_item",
			Description: "A new item has been received through the Feed",
			Options: []bees.PlaceholderDescriptor{
				{
					Name:        "title",
					Description: "Title of the Item",
					Type:        "string",
				},
				{
					Name:        "description",
					Description: "Description of the Item",
					Type:        "string",
				},
				{
					Name:        "url",
					Description: "URL of the Item",
					Type:        "string",
				},
			},
		},
	}
	return events
}

func init() {
	f := CrawlerBeeFactory{}
	bees.RegisterFactory(&f)
}
