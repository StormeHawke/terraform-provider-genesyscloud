// Package wfm holds the helpers that the workforce management resources share.
package wfm

import (
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
)

// GetAllBusinessUnits retrieves every business unit in the organization.
// The empty strings are for feature (so no special permission checking overrides) and divisionId (so no filtering by division).
func GetAllBusinessUnits(api *platformclientv2.WorkforceManagementApi) (*[]platformclientv2.Businessunitlistitem, *platformclientv2.APIResponse, error) {
	listing, resp, err := api.GetWorkforcemanagementBusinessunits("", "")
	if err != nil {
		return nil, resp, err
	}

	return listing.Entities, resp, nil
}

// GetAllManagementUnits retrieves every management unit in the organization, across all business units.
// The api documents paging as unsupported and returns every management unit, so the paging arguments are placeholders.
func GetAllManagementUnits(api *platformclientv2.WorkforceManagementApi) (*[]platformclientv2.Managementunit, *platformclientv2.APIResponse, error) {
	listing, resp, err := api.GetWorkforcemanagementManagementunits(100, 1, "", "", "")
	if err != nil {
		return nil, resp, err
	}

	return listing.Entities, resp, nil
}
