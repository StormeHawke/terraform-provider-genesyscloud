package workforcemanagement_managementunits

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/consistency_checker"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/provider"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util/constants"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util/resourcedata"
)

/*
The resource_genesyscloud_workforcemanagement_managementunits.go contains all of the methods that perform the core logic for a resource.
*/

// getAllManagementUnits retrieves all of the management units via Terraform in the Genesys Cloud and is used for the exporter
func getAllManagementUnits(ctx context.Context, clientConfig *platformclientv2.Configuration) (resourceExporter.ResourceIDMetaMap, diag.Diagnostics) {
	proxy := getManagementUnitProxy(clientConfig)
	resources := make(resourceExporter.ResourceIDMetaMap)

	managementUnits, resp, err := proxy.getAllManagementUnits(ctx)
	if err != nil {
		return nil, util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to get management units: %v", err), resp)
	}

	for _, managementUnit := range *managementUnits {
		if managementUnit.Id == nil || managementUnit.Name == nil {
			continue
		}
		resources[*managementUnit.Id] = &resourceExporter.ResourceMeta{BlockLabel: *managementUnit.Name}
	}

	return resources, nil
}

// createManagementUnit is used by the workforcemanagement_managementunits resource to create a Genesys Cloud management unit
func createManagementUnit(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getManagementUnitProxy(sdkConfig)
	managementUnit := buildCreateManagementUnitRequest(d)

	log.Printf("Creating management unit %s", *managementUnit.Name)
	created, resp, err := proxy.createManagementUnit(ctx, managementUnit)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to create management unit %s: %s", *managementUnit.Name, err), resp)
	}

	d.SetId(*created.Id)
	log.Printf("Created management unit %s %s", *created.Id, *managementUnit.Name)
	return readManagementUnit(ctx, d, meta)
}

// readManagementUnit is used by the workforcemanagement_managementunits resource to read a management unit from Genesys Cloud
func readManagementUnit(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getManagementUnitProxy(sdkConfig)
	cc := consistency_checker.NewConsistencyCheck(ctx, d, meta, ResourceManagementUnit(), constants.ConsistencyChecks(), ResourceType)

	log.Printf("Reading management unit %s", d.Id())

	return util.WithRetriesForRead(ctx, d, func() *retry.RetryError {
		managementUnit, resp, getErr := proxy.getManagementUnitById(ctx, d.Id())
		if getErr != nil {
			if util.IsStatus404(resp) {
				return retry.RetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s: %s", d.Id(), getErr), resp))
			}
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s: %s", d.Id(), getErr), resp))
		}

		resourcedata.SetNillableValue(d, "name", managementUnit.Name)
		if managementUnit.BusinessUnit != nil {
			resourcedata.SetNillableValue(d, "business_unit_id", managementUnit.BusinessUnit.Id)
		}
		if managementUnit.Division != nil {
			resourcedata.SetNillableValue(d, "division_id", managementUnit.Division.Id)
		}
		_ = d.Set("settings", flattenSettings(managementUnit.Settings))

		log.Printf("Read management unit %s", d.Id())
		return cc.CheckState(d)
	})
}

// updateManagementUnit is used by the workforcemanagement_managementunits resource to update a management unit in Genesys Cloud
func updateManagementUnit(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getManagementUnitProxy(sdkConfig)

	log.Printf("Updating management unit %s", d.Id())

	// Changing business_unit_id moves the management unit through the move api instead of dropping and recreating it, because
	// recreating would destroy a management unit that may have agents, schedules and other configuration attached to it.
	// The move is asynchronous, so the business unit is waited on before the rest of the update is applied.
	if d.HasChange("business_unit_id") {
		if diagErr := moveManagementUnitToBusinessUnit(ctx, proxy, d.Id(), d.Get("business_unit_id").(string)); diagErr != nil {
			return diagErr
		}
	}

	// The update request carries the current settings metadata version for optimistic locking, so a version conflict re-reads the management unit and tries again
	diagErr := util.WithRetries(ctx, 60*time.Second, func() *retry.RetryError {
		current, resp, err := proxy.getManagementUnitById(ctx, d.Id())
		if err != nil {
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s before update: %s", d.Id(), err), resp))
		}

		var metadata *platformclientv2.Wfmversionedentitymetadata
		if current.Settings != nil && current.Settings.Metadata != nil {
			metadata = current.Settings.Metadata
		} else {
			metadata = current.Metadata
		}

		_, resp, err = proxy.updateManagementUnit(ctx, d.Id(), buildUpdateManagementUnitRequest(d, metadata))
		if err == nil {
			return nil
		}
		builtDiagErr := util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to update management unit %s: %s", d.Id(), err), resp)
		if util.IsStatus409(resp) {
			return retry.RetryableError(builtDiagErr)
		}
		return retry.NonRetryableError(builtDiagErr)
	})
	if diagErr != nil {
		return diagErr
	}

	log.Printf("Updated management unit %s", d.Id())
	return readManagementUnit(ctx, d, meta)
}

// moveManagementUnitToBusinessUnit moves the management unit to another business unit. The move is processed asynchronously,
// so it waits until the management unit reports the new business unit.
func moveManagementUnitToBusinessUnit(ctx context.Context, proxy *managementUnitProxy, managementUnitId string, businessUnitId string) diag.Diagnostics {
	log.Printf("Moving management unit %s to business unit %s", managementUnitId, businessUnitId)
	_, resp, err := proxy.moveManagementUnit(ctx, managementUnitId, businessUnitId)
	if err != nil {
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to move management unit %s to business unit %s: %s", managementUnitId, businessUnitId, err), resp)
	}

	return util.WithRetries(ctx, 5*time.Minute, func() *retry.RetryError {
		managementUnit, resp, err := proxy.getManagementUnitById(ctx, managementUnitId)
		if err != nil {
			return retry.NonRetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("Failed to read management unit %s while waiting for the move: %s", managementUnitId, err), resp))
		}
		if managementUnit.BusinessUnit != nil && managementUnit.BusinessUnit.Id != nil && *managementUnit.BusinessUnit.Id == businessUnitId {
			log.Printf("Moved management unit %s to business unit %s", managementUnitId, businessUnitId)
			return nil
		}
		return retry.RetryableError(util.BuildWithRetriesApiDiagnosticError(ResourceType, fmt.Sprintf("management unit %s has not moved to business unit %s yet", managementUnitId, businessUnitId), resp))
	})
}

// deleteManagementUnit is used by the workforcemanagement_managementunits resource to delete a management unit from Genesys Cloud
func deleteManagementUnit(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*provider.ProviderMeta).ClientConfig
	proxy := getManagementUnitProxy(sdkConfig)

	log.Printf("Deleting management unit %s", d.Id())
	resp, err := proxy.deleteManagementUnit(ctx, d.Id())
	if err != nil {
		if util.IsStatus404(resp) {
			log.Printf("Management unit %s already deleted", d.Id())
			return nil
		}
		return util.BuildAPIDiagnosticError(ResourceType, fmt.Sprintf("Failed to delete management unit %s: %s", d.Id(), err), resp)
	}

	log.Printf("Deleted management unit %s", d.Id())
	return nil
}
