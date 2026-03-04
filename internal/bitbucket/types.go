package bitbucket

import "time"

type User struct {
	DisplayName string `json:"display_name"`
	UUID        string `json:"uuid"`
	Nickname    string `json:"nickname"`
	AccountID   string `json:"account_id"`
}

type Branch struct {
	Name string `json:"name"`
}

type BranchRef struct {
	Branch     Branch     `json:"branch"`
	Repository Repository `json:"repository"`
	Commit     Commit     `json:"commit"`
}

type Repository struct {
	FullName string `json:"full_name"`
	Name     string `json:"name"`
	UUID     string `json:"uuid"`
}

type Commit struct {
	Hash string `json:"hash"`
}

type Links struct {
	HTML Link `json:"html"`
	Diff Link `json:"diff"`
	Self Link `json:"self"`
}

type Link struct {
	Href string `json:"href"`
}

type PullRequest struct {
	ID                  int         `json:"id"`
	Title               string      `json:"title"`
	Description         string      `json:"description"`
	State               string      `json:"state"`
	Author              User        `json:"author"`
	Source              BranchRef   `json:"source"`
	Destination         BranchRef   `json:"destination"`
	CloseSourceBranch   bool        `json:"close_source_branch"`
	CreatedOn           time.Time   `json:"created_on"`
	UpdatedOn           time.Time   `json:"updated_on"`
	Reviewers           []User      `json:"reviewers"`
	Participants        []Participant `json:"participants"`
	Links               Links       `json:"links"`
	CommentCount        int         `json:"comment_count"`
	TaskCount           int         `json:"task_count"`
}

type Participant struct {
	User     User   `json:"user"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
	State    string `json:"state"`
}

type Comment struct {
	ID        int       `json:"id"`
	Content   Content   `json:"content"`
	User      User      `json:"user"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
	Inline    *Inline   `json:"inline,omitempty"`
	Deleted   bool      `json:"deleted"`
	Links     Links     `json:"links"`
}

type Content struct {
	Raw    string `json:"raw"`
	Markup string `json:"markup"`
	HTML   string `json:"html"`
}

type Inline struct {
	Path string `json:"path"`
	From *int   `json:"from"`
	To   *int   `json:"to"`
}

type DiffStat struct {
	Status   string     `json:"status"`
	Old      *DiffFile  `json:"old"`
	New      *DiffFile  `json:"new"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
}

type DiffFile struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type RepoInfo struct {
	FullName     string    `json:"full_name"`
	Name         string    `json:"name"`
	MainBranch   *Branch   `json:"mainbranch"`
	Links        Links     `json:"links"`
}

type PaginatedResponse[T any] struct {
	Size     int    `json:"size"`
	Page     int    `json:"page"`
	PageLen  int    `json:"pagelen"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Values   []T    `json:"values"`
}

type Workspace struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
