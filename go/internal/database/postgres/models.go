package postgres

import (
	"slices"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

type Vulnerability struct {
	ID          string         `gorm:"primaryKey"`
	SourceID    string         `gorm:"index;not null"`
	Modified    time.Time      `gorm:"not null"`
	IsWithdrawn bool           `gorm:"not null;default:false"`
	ModifiedRaw time.Time      `gorm:"not null"`
	AliasRaw    pq.StringArray `gorm:"type:text[]"`
	RelatedRaw  pq.StringArray `gorm:"type:text[]"`
	UpstreamRaw pq.StringArray `gorm:"type:text[]"`
}

type AffectedEvent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type AffectedVersions struct {
	ID        uint           `gorm:"primaryKey"`
	VulnID    string         `gorm:"index;not null"`
	Ecosystem string         `gorm:"index;not null"`
	Name      string         `gorm:"index;not null"`
	Versions  pq.StringArray `gorm:"type:text[]"`
	Events    datatypes.JSON `gorm:"type:jsonb"` // Stores []AffectedEvent as JSONB
	CoarseMin string         `gorm:""`
	CoarseMax string         `gorm:""`
}

func (av AffectedVersions) SortKey() string {
	var b strings.Builder
	b.WriteString(av.VulnID)
	b.WriteString("\x1f")
	b.WriteString(av.Ecosystem)
	b.WriteString("\x1f")
	b.WriteString(av.Name)
	b.WriteString("\x1f")

	sortedVersions := slices.Clone(av.Versions)
	slices.Sort(sortedVersions)
	b.WriteString(strings.Join(sortedVersions, ","))
	b.WriteString("\x1f")

	// Note: If Events are stored as JSONB, you would unmarshal them here if needed for sorting,
	// or handle them based on how you project your struct logic.
	return b.String()
}

type AffectedCommits struct {
	ID      uint           `gorm:"primaryKey"`
	BugID   string         `gorm:"column:bug_id;index;not null"`
	Commits datatypes.JSON `gorm:"type:jsonb"` // Stored safely as JSON array of byte slices
	Public  bool           `gorm:"not null;default:false"`
	Page    int            `gorm:"not null;default:0"`
}

type AliasGroup struct {
	ID           uint           `gorm:"primaryKey"`
	BugIDs       pq.StringArray `gorm:"type:text[]"`
	LastModified time.Time      `gorm:"column:last_modified;not null"`
}

type UpstreamGroup struct {
	ID                uint           `gorm:"primaryKey"`
	DBID              string         `gorm:"column:db_id;index;not null"`
	UpstreamIDs       pq.StringArray `gorm:"type:text[]"`
	LastModified      time.Time      `gorm:"column:last_modified;not null"`
	UpstreamHierarchy []byte         `gorm:"type:bytea"`
}

type RelatedGroup struct {
	ID         uint           `gorm:"primaryKey"`
	RelatedIDs pq.StringArray `gorm:"type:text[]"`
	Modified   time.Time      `gorm:"not null"`
}

type AliasAllowListEntry struct {
	ID    uint   `gorm:"primaryKey"`
	BugID string `gorm:"column:bug_id;uniqueIndex;not null"`
}

type AliasDenyListEntry struct {
	ID    uint   `gorm:"primaryKey"`
	BugID string `gorm:"column:bug_id;uniqueIndex;not null"`
}

type RepoAllowList struct {
	ID                    uint   `gorm:"primaryKey"`
	Type                  string `gorm:"not null"` // 'url' or 'regex'
	Value                 string `gorm:"not null"` // normalized URL or regex pattern
	ConsiderAllBranches   bool   `gorm:"not null;default:false"`
	CherrypicksIntroduced bool   `gorm:"not null;default:false"`
	CherrypicksFixed      bool   `gorm:"not null;default:false"`
	CherrypicksLimit      bool   `gorm:"not null;default:false"`
}

type Severity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

type ListedVulnerability struct {
	ID               uint           `gorm:"primaryKey"`
	Published        time.Time      `gorm:"index;not null"`
	Ecosystems       pq.StringArray `gorm:"type:text[]"`
	Packages         pq.StringArray `gorm:"type:text[]"`
	Summary          string         `gorm:"type:text"`
	IsFixed          bool           `gorm:"not null;default:false"`
	Severities       datatypes.JSON `gorm:"type:jsonb"` // Stores []Severity as JSONB
	AutocompleteTags pq.StringArray `gorm:"type:text[]"`
	SearchIndices    pq.StringArray `gorm:"type:text[]"`
}

func (l *ListedVulnerability) Equal(other *ListedVulnerability) bool {
	if l == nil || other == nil {
		return l == other
	}

	return l.Published.Equal(other.Published) &&
		l.Summary == other.Summary &&
		l.IsFixed == other.IsFixed &&
		slices.Equal(l.Ecosystems, other.Ecosystems) &&
		slices.Equal(l.Packages, other.Packages) &&
		slices.Equal(l.AutocompleteTags, other.AutocompleteTags) &&
		slices.Equal(l.SearchIndices, other.SearchIndices)
}

type SourceRepository struct {
	ID                      uint           `gorm:"primaryKey"`
	Type                    string         `gorm:"not null"`
	Name                    string         `gorm:"uniqueIndex;not null"`
	RepoURL                 string         `gorm:""`
	RepoUsername            string         `gorm:""`
	RepoBranch              string         `gorm:""`
	RESTAPIURL              string         `gorm:""`
	Bucket                  string         `gorm:""`
	DirectoryPath           string         `gorm:""`
	LastSyncedHash          string         `gorm:""`
	LastUpdateDate          *time.Time     `gorm:""`
	IgnorePatterns          pq.StringArray `gorm:"type:text[]"`
	Editable                bool           `gorm:"not null;default:false"`
	Extension               string         `gorm:""`
	KeyPath                 string         `gorm:""`
	AcceptedEcosystems      pq.StringArray `gorm:"type:text[]"`
	IgnoreGit               bool           `gorm:"not null;default:false"`
	DetectCherrypicks       bool           `gorm:"not null;default:false"`
	ConsiderAllBranches     bool           `gorm:"not model;default:false"`
	VersionsFromRepo        bool           `gorm:"not null;default:false"`
	IgnoreLastImportTime    bool           `gorm:"not null;default:false"`
	IgnoreDeletionThreshold bool           `gorm:"not null;default:false"`
	Link                    string         `gorm:""`
	HumanLink               string         `gorm:""`
	DBPrefix                pq.StringArray `gorm:"type:text[]"`
	StrictValidation        bool           `gorm:"not null;default:false"`
	WorkPool                string         `gorm:""`
}

type RepoIndex struct {
	ID                uint           `gorm:"primaryKey"`
	Name              string         `gorm:"index;not null"`
	BaseCPE           string         `gorm:""`
	Commit            []byte         `gorm:"type:bytea"`
	Tag               string         `gorm:""`
	When              time.Time      `gorm:"column:repo_when"`
	RepoType          string         `gorm:""`
	RepoAddr          string         `gorm:""`
	FileExts          pq.StringArray `gorm:"type:text[]"`
	FileHashType      string         `gorm:""`
	EmptyBucketBitmap []byte         `gorm:"type:bytea"`
	FileCount         int            `gorm:"not null;default:0"`
	DocumentVersion   int            `gorm:"not null;default:0"`
}

type RepoIndexBucket struct {
	ID              uint   `gorm:"primaryKey"`
	NodeHash        []byte `gorm:"type:bytea;not null"`
	FilesContained  int    `gorm:"not null;default:0"`
	DocumentVersion int    `gorm:"not null;default:0"`
}
