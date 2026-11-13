locals {
  dependencies = {
    resource = [
      "../genesyscloud_workforcemanagement_businessunits/resource.tf",
      "../genesyscloud_workforcemanagement_businessunits_servicegoaltemplate/resource.tf",
      "../genesyscloud_routing_queue/resource.tf",
      "../genesyscloud_routing_language/resource.tf",
      "../genesyscloud_routing_skill/resource.tf",
    ]
  }
}
