package workforcemanagement_businessunits_planninggroup

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mypurecloud/platform-client-sdk-go/v200/platformclientv2"
	resourceExporter "github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/resource_exporter"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/util/lists"
)

// SplitPlanningGroupTerraformId splits the resource id, which is in the form
// <businessUnitId>/<planningGroupId>, into the businessUnitId and planningGroupId
func SplitPlanningGroupTerraformId(id string) (businessUnitId string, planningGroupId string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return id, ""
	}
	return parts[0], parts[1]
}

// ModifyServiceGoalTemplateIdStateValue strips the business unit from a service goal template id before it is saved in state.
// The service goal template resource id is <businessUnitId>/<serviceGoalTemplateId>, but the planning group api only returns the
// template id, so saving the composite value would cause a non-empty plan.
// Stripping the business unit is safe because a planning group can only use a service goal template from its own business unit.
func ModifyServiceGoalTemplateIdStateValue(id interface{}) string {
	templateId := id.(string)
	if i := strings.LastIndex(templateId, "/"); i >= 0 {
		return templateId[i+1:]
	}
	return templateId
}

// ServiceGoalTemplateRefResolver resolves a bare service goal template id to a Terraform reference by searching the
// service goal template exporter's resource map for a composite key ending with "/<serviceGoalTemplateId>".
func ServiceGoalTemplateRefResolver(attrName string) func(configMap map[string]interface{}, exporters map[string]*resourceExporter.ResourceExporter, resourceLabel string) error {
	return func(configMap map[string]interface{}, exporters map[string]*resourceExporter.ResourceExporter, resourceLabel string) error {
		templateId, ok := configMap[attrName].(string)
		if !ok || templateId == "" || strings.HasPrefix(templateId, "${") {
			return nil
		}

		exporter := exporters[serviceGoalTemplateResourceType]
		if exporter == nil || exporter.SanitizedResourceMap == nil {
			return nil
		}

		suffix := "/" + templateId
		for compositeId, meta := range exporter.SanitizedResourceMap {
			if strings.HasSuffix(compositeId, suffix) && meta != nil && meta.BlockLabel != "" {
				configMap[attrName] = fmt.Sprintf("${%s.%s.id}", serviceGoalTemplateResourceType, meta.BlockLabel)
				return nil
			}
		}
		return nil
	}
}

// buildRoutePaths intentionally never sets sourcePlanningGroup. The api uses that field to take a route path away from another
// planning group, which silently modifies a planning group this resource does not manage. This resource is configuration as code,
// so a route path that already belongs to another planning group must be removed from it explicitly, and the api error surfaces otherwise.
func buildRoutePaths(d *schema.ResourceData) *[]platformclientv2.Routepathrequest {
	routePaths := make([]platformclientv2.Routepathrequest, 0)
	routePathSet, ok := d.Get("route_paths").(*schema.Set)
	if !ok {
		return &routePaths
	}

	for _, item := range routePathSet.List() {
		routePathMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		routePath := platformclientv2.Routepathrequest{
			QueueId:   platformclientv2.String(routePathMap["queue_id"].(string)),
			MediaType: platformclientv2.String(routePathMap["media_type"].(string)),
		}
		if languageId, ok := routePathMap["language_id"].(string); ok && languageId != "" {
			routePath.LanguageId = &languageId
		}
		if skillSet, ok := routePathMap["skill_ids"].(*schema.Set); ok && skillSet.Len() > 0 {
			routePath.SkillIds = lists.SetToStringList(skillSet)
		}
		routePaths = append(routePaths, routePath)
	}
	return &routePaths
}

func buildServiceGoalTemplateId(d *schema.ResourceData) *string {
	templateId, ok := d.Get("service_goal_template_id").(string)
	if !ok || templateId == "" {
		return nil
	}
	templateId = ModifyServiceGoalTemplateIdStateValue(templateId)
	return &templateId
}

func buildCreatePlanningGroupRequest(d *schema.ResourceData) *platformclientv2.Createplanninggrouprequest {
	return &platformclientv2.Createplanninggrouprequest{
		Name:                  platformclientv2.String(d.Get("name").(string)),
		RoutePaths:            buildRoutePaths(d),
		ServiceGoalTemplateId: buildServiceGoalTemplateId(d),
	}
}

func buildUpdatePlanningGroupRequest(d *schema.ResourceData, metadata *platformclientv2.Wfmversionedentitymetadata) *platformclientv2.Updateplanninggrouprequest {
	request := &platformclientv2.Updateplanninggrouprequest{
		Name:                  platformclientv2.String(d.Get("name").(string)),
		RoutePaths:            &platformclientv2.Setwrapperroutepathrequest{Values: buildRoutePaths(d)},
		ServiceGoalTemplateId: buildServiceGoalTemplateId(d),
	}
	if metadata != nil && metadata.Version != nil {
		request.Metadata = &platformclientv2.Wfmversionedentitymetadata{Version: metadata.Version}
	}
	return request
}

func flattenServiceGoalTemplateId(templateRef *platformclientv2.Servicegoaltemplatereference) string {
	if templateRef == nil || templateRef.Id == nil {
		return ""
	}
	return *templateRef.Id
}

func flattenRoutePaths(routePaths *[]platformclientv2.Routepathresponse) *schema.Set {
	flattened := make([]interface{}, 0)
	if routePaths != nil {
		for _, routePath := range *routePaths {
			routePathMap := make(map[string]interface{})
			if routePath.Queue != nil && routePath.Queue.Id != nil {
				routePathMap["queue_id"] = *routePath.Queue.Id
			}
			if routePath.MediaType != nil {
				routePathMap["media_type"] = *routePath.MediaType
			}
			if routePath.Language != nil && routePath.Language.Id != nil {
				routePathMap["language_id"] = *routePath.Language.Id
			}
			skillIds := make([]interface{}, 0)
			if routePath.Skills != nil {
				for _, skill := range *routePath.Skills {
					if skill.Id != nil {
						skillIds = append(skillIds, *skill.Id)
					}
				}
			}
			routePathMap["skill_ids"] = schema.NewSet(schema.HashString, skillIds)
			flattened = append(flattened, routePathMap)
		}
	}
	return schema.NewSet(schema.HashResource(routePathResource), flattened)
}

// GeneratePlanningGroupResource returns terraform config for a planning group, used in tests
func GeneratePlanningGroupResource(resourceLabel string, businessUnitId string, name string, attrs ...string) string {
	return fmt.Sprintf(`resource "%s" "%s" {
	business_unit_id = %s
	name             = "%s"
	%s
}
`, ResourceType, resourceLabel, businessUnitId, name, strings.Join(attrs, "\n"))
}
