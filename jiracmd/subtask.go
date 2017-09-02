package jiracmd

import (
	"fmt"

	"github.com/coryb/figtree"
	"github.com/coryb/oreo"

	"gopkg.in/Netflix-Skunkworks/go-jira.v1"
	"gopkg.in/Netflix-Skunkworks/go-jira.v1/jiracli"
	"gopkg.in/Netflix-Skunkworks/go-jira.v1/jiradata"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type SubtaskOptions struct {
	jiracli.GlobalOptions        `yaml:",inline" json:",inline" figtree:",inline"`
	jiradata.IssueUpdate `yaml:",inline" json:",inline" figtree:",inline"`
	Project              string            `yaml:"project,omitempty" json:"project,omitempty"`
	IssueType            string            `yaml:"issuetype,omitempty" json:"issuetype,omitempty"`
	Overrides            map[string]string `yaml:"overrides,omitempty" json:"overrides,omitempty"`
	Issue                string            `yaml:"issue,omitempty" json:"issue,omitempty"`
}

func CmdSubtaskRegistry(fig *figtree.FigTree, o *oreo.Client) *jiracli.CommandRegistryEntry {
	opts := SubtaskOptions{
		GlobalOptions: jiracli.GlobalOptions{
			Template: figtree.NewStringOption("subtask"),
		},
		Overrides: map[string]string{},
	}

	return &jiracli.CommandRegistryEntry{
		"Subtask issue",
		func() error {
			return CmdSubtask(o, &opts)
		},
		func(cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			if opts.IssueType == "" {
				opts.IssueType = "Sub-task"
			}
			return CmdSubtaskUsage(cmd, &opts)
		},
	}
}

func CmdSubtaskUsage(cmd *kingpin.CmdClause, opts *SubtaskOptions) error {
	if err := jiracli.GlobalUsage(cmd, &opts.GlobalOptions); err != nil {
		return err
	}
	jiracli.BrowseUsage(cmd, &opts.GlobalOptions)
	jiracli.EditorUsage(cmd, &opts.GlobalOptions)
	jiracli.TemplateUsage(cmd, &opts.GlobalOptions)
	cmd.Flag("noedit", "Disable opening the editor").SetValue(&opts.SkipEditing)
	cmd.Flag("project", "project to subtask issue in").Short('p').StringVar(&opts.Project)
	cmd.Flag("comment", "Comment message for issue").Short('m').PreAction(func(ctx *kingpin.ParseContext) error {
		opts.Overrides["comment"] = jiracli.FlagValue(ctx, "comment")
		return nil
	}).String()
	cmd.Flag("override", "Set issue property").Short('o').StringMapVar(&opts.Overrides)
	cmd.Arg("ISSUE", "Parent issue for subtask").StringVar(&opts.Issue)
	return nil
}

// CmdSubtask sends the subtask-metadata to the "subtask" template for editing, then
// will parse the edited document as YAML and submit the document to jira.
func CmdSubtask(o *oreo.Client, opts *SubtaskOptions) error {
	type templateInput struct {
		Meta      *jiradata.IssueType `yaml:"meta" json:"meta"`
		Overrides map[string]string   `yaml:"overrides" json:"overrides"`
		Parent    *jiradata.Issue     `yaml:"parent" json:"parent"`
	}

	parent, err := jira.GetIssue(o, opts.Endpoint.Value, opts.Issue, nil)
	if err != nil {
		return err
	}

	if project, ok := parent.Fields["project"].(map[string]interface{}); ok {
		if key, ok := project["key"].(string); ok {
			opts.Project = key
		} else {
			return fmt.Errorf("Failed to find Project Key in parent issue")
		}
	} else {
		return fmt.Errorf("Failed to find Project field in parent issue")
	}

	createMeta, err := jira.GetIssueCreateMetaIssueType(o, opts.Endpoint.Value, opts.Project, opts.IssueType)
	if err != nil {
		return err
	}

	issueUpdate := jiradata.IssueUpdate{}
	input := templateInput{
		Meta:      createMeta,
		Overrides: opts.Overrides,
		Parent:    parent,
	}
	input.Overrides["project"] = opts.Project
	input.Overrides["issuetype"] = opts.IssueType
	input.Overrides["user"] = opts.User.Value

	var issueResp *jiradata.IssueCreateResponse
	err = jiracli.EditLoop(&opts.GlobalOptions, &input, &issueUpdate, func() error {
		issueResp, err = jira.CreateIssue(o, opts.Endpoint.Value, &issueUpdate)
		return err
	})
	if err != nil {
		return err
	}

	fmt.Printf("OK %s %s/browse/%s\n", issueResp.Key, opts.Endpoint.Value, issueResp.Key)

	if opts.Browse.Value {
		return CmdBrowse(&BrowseOptions{opts.GlobalOptions, issueResp.Key})
	}
	return nil
}