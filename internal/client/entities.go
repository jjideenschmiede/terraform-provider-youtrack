// Copyright 2026 J&J Ideenschmiede GmbH
// SPDX-License-Identifier: MPL-2.0

package client

type Ref struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"$type,omitempty"`
}

type NotificationTemplate struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

const NotificationTemplateFields = "id,content"

type User struct {
	ID       string `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	RingID   string `json:"ringId"`
	Banned   bool   `json:"banned"`
	Guest    bool   `json:"guest"`
}

const UserFields = "id,login,fullName,email,ringId,banned,guest"

type UserGroup struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	RingID     string `json:"ringId"`
	UsersCount int64  `json:"usersCount"`
}

const UserGroupFields = "id,name,ringId,usersCount"

type Project struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ShortName    string `json:"shortName"`
	Description  string `json:"description"`
	Archived     bool   `json:"archived"`
	Template     bool   `json:"template"`
	FromEmail    string `json:"fromEmail"`
	ReplyToEmail string `json:"replyToEmail"`
	Leader       *Ref   `json:"leader"`
}

const ProjectFields = "id,name,shortName,description,archived,template,fromEmail,replyToEmail,leader(id)"

type FieldType struct {
	ID           string `json:"id"`
	ValueType    string `json:"valueType"`
	IsMultiValue bool   `json:"isMultiValue"`
}

type CustomField struct {
	ID                     string     `json:"id"`
	Name                   string     `json:"name"`
	Aliases                string     `json:"aliases"`
	IsAutoAttached         bool       `json:"isAutoAttached"`
	IsDisplayedInIssueList bool       `json:"isDisplayedInIssueList"`
	FieldType              *FieldType `json:"fieldType"`
}

const CustomFieldFields = "id,name,aliases,isAutoAttached,isDisplayedInIssueList,fieldType(id,valueType,isMultiValue)"

type FieldStyle struct {
	ID string `json:"id"`
}

type BundleElement struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Archived    bool        `json:"archived"`
	Ordinal     int64       `json:"ordinal"`
	IsResolved  bool        `json:"isResolved"`
	Color       *FieldStyle `json:"color"`
}

type Bundle struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Values []BundleElement `json:"values"`
}

const BundleElementFields = "id,name,description,archived,ordinal,isResolved,color(id)"

const BundleFields = "id,name,values(" + BundleElementFields + ")"

type ProjectCustomField struct {
	ID             string       `json:"id"`
	Type           string       `json:"$type"`
	CanBeEmpty     bool         `json:"canBeEmpty"`
	EmptyFieldText string       `json:"emptyFieldText"`
	IsPublic       bool         `json:"isPublic"`
	Ordinal        int64        `json:"ordinal"`
	Field          *CustomField `json:"field"`
	Bundle         *Ref         `json:"bundle"`
}

const ProjectCustomFieldFields = "id,$type,canBeEmpty,emptyFieldText,isPublic,ordinal,field(id,fieldType(id)),bundle(id)"

type Tag struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	UntagOnResolve bool        `json:"untagOnResolve"`
	Color          *FieldStyle `json:"color"`
	VisibleFor     *Ref        `json:"visibleFor"`
	UpdateableBy   *Ref        `json:"updateableBy"`
}

const TagFields = "id,name,untagOnResolve,color(id),visibleFor(id),updateableBy(id)"

type SavedQuery struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Query        string `json:"query"`
	VisibleFor   *Ref   `json:"visibleFor"`
	UpdateableBy *Ref   `json:"updateableBy"`
}

const SavedQueryFields = "id,name,query,visibleFor(id),updateableBy(id)"

type IssueLinkType struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SourceToTarget string `json:"sourceToTarget"`
	TargetToSource string `json:"targetToSource"`
	Directed       bool   `json:"directed"`
	Aggregation    bool   `json:"aggregation"`
	ReadOnly       bool   `json:"readOnly"`
}

const IssueLinkTypeFields = "id,name,sourceToTarget,targetToSource,directed,aggregation,readOnly"
