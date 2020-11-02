package model

type Data struct {
	Abstract string   `json:"abstract,omitempty"`
	Keywords []string `json:"keywords,omitempty"`
	DOI      string   `json:"doi,omitempty"`
	Assembly Assembly `json:"assembly,omitempty"`
}

type Assembly struct {
	Url   string  `json:"url,omitempty"`
	Links []Link `json:"links,omitempty"`
}

type Link struct {
	Url    string  `json:"url,omitempty"`
	Report Report `json:"report,omitempty"`
}

type Report struct {
	OrganismName      string     `json:"organismName,omitempty"`
	TaxonomyUrl       string     `json:"taxonomyUrl,omitempty"`
	InfraspecificName string     `json:"infraspecificName,omitempty"`
	BioSample         BioSample `json:"bioSample,omitempty"`
	Submitter         string     `json:"submitter,omitempty"`
	Date              string     `json:"date,omitempty"`
	FTPUrl            string     `json:"ftpUrl,omitempty"`
	GBFFUrl           string     `json:"gbffUrl,omitempty"`
	GBFFPath          string     `json:"gbffPath,omitempty"`
}

type BioSample struct {
	Url                            string `json:"url,omitempty"`
	Strain                         string `json:"strain,omitempty"`
	CollectionDate                 string `json:"collectionDate,omitempty"`
	BroadScaleEnvironmentalContext string `json:"broadScaleEnvironmentalContext,omitempty"`
	LocalScaleEnvironmentalContext string `json:"localScaleEnvironmentalContext,omitempty"`
	EnvironmentalMedium            string `json:"environmentalMedium,omitempty"`
	GeographicLocation             string `json:"geographicLocation,omitempty"`
	LatLong                        string `json:"latLong,omitempty"`
	Host                           string `json:"host,omitempty"`
	IsolationAndGrowthCondition    string `json:"isolationAndGrowthCondition,omitempty"`
	NumberOfReplicons              string `json:"numberOfReplicons,omitempty"`
	Ploidy                         string `json:"ploidy,omitempty"`
	Propagation                    string `json:"propagation,omitempty"`
}
