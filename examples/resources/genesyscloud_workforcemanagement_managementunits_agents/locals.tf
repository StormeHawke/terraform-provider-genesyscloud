locals {
  dependencies = {
    resource = [
      "../genesyscloud_workforcemanagement_managementunits/resource.tf",
      "../genesyscloud_user/resource.tf",
    ]
  }
}
