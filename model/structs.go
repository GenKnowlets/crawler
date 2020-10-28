package model

type Data struct {
	Abstract string   `json:"abstract"`
	Keywords []string `json:"keywords"`
	DOI      string   `json:"doi"`
	Assembly Assembly `json:"assembly"`
}

type Assembly struct {
	Url   string  `json:"url"`
	Links []*Link `json:"links"`
}

type Link struct {
	Url    string  `json:"url"`
	Report *Report `json:"report"`
}

type Report struct {
	OrganismName      string     `json:"organismName"`
	TaxonomyUrl       string     `json:"taxonomyUrl"`
	InfraspecificName string     `json:"infraspecificName"`
	BioSample         *BioSample `json:"bioSample"`
	Submitter         string     `json:"submitter"`
	Date              string     `json:"date"`
	FTPUrl            string     `json:"ftpUrl"`
	GBFFUrl           string     `json:"gbffUrl"`
}

type BioSample struct {
	Url                            string `json:"url"`
	Strain                         string `json:"strain"`
	CollectionDate                 string `json:"collectionDate"`
	BroadScaleEnvironmentalContext string `json:"broadScaleEnvironmentalContext"`
	LocalScaleEnvironmentalContext string `json:"localScaleEnvironmentalContext"`
	EnvironmentalMedium            string `json:"environmentalMedium"`
	GeographicLocation             string `json:"geographicLocation"`
	LatLong                        string `json:"latLong"`
	Host                           string `json:"host"`
	IsolationAndGrowthCondition    string `json:"isolationAndGrowthCondition"`
	NumberOfReplicons              string `json:"numberOfReplicons"`
	Ploidy                         string `json:"ploidy"`
	Propagation                    string `json:"propagation"`
}
