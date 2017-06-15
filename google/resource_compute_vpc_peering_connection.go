package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	compute "google.golang.org/api/compute/v1"
)

func resourceComputeVpcPeeringConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeVpcPeeringConnectionCreate,
		Read:   resourceComputeVpcPeeringConnectionRead,
		Delete: resourceComputeVpcPeeringConnectionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"peer_network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"auto_create_routes": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"state_details": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeVpcPeeringConnectionNetwork(d *schema.ResourceData) (string, error) {
	network, err := getNetworkName(d, "network")
	if err != nil {
		return "", err
	} else if network == "" {
		return "", fmt.Errorf("%q: required field is not set", "network")
	}
	return network, nil
}

func resourceComputeVpcPeeringConnectionCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	network, err := resourceComputeVpcPeeringConnectionNetwork(d)
	if err != nil {
		return err
	}

	// Build the add-peering request
	peering := &compute.NetworksAddPeeringRequest{
		Name:             d.Get("name").(string),
		PeerNetwork:      d.Get("peer_network").(string),
		AutoCreateRoutes: d.Get("auto_create_routes").(bool),
		ForceSendFields:  []string{"AutoCreateRoutes"},
	}

	log.Printf("[DEBUG] Network add peering request: %#v", network)
	op, err := config.clientCompute.Networks.AddPeering(
		project, network, peering).Do()
	if err != nil {
		return fmt.Errorf("Error creating network peering: %s", err)
	}

	d.SetId(peering.Name)

	err = computeOperationWaitGlobal(config, op, project, "Creating Network Peering")
	if err != nil {
		return err
	}

	return resourceComputeVpcPeeringConnectionRead(d, meta)
}

func resourceComputeVpcPeeringConnectionRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	networkName, err := resourceComputeVpcPeeringConnectionNetwork(d)
	if err != nil {
		return err
	}

	network, err := config.clientCompute.Networks.Get(
		project, networkName).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Network %q", d.Get("name").(string)))
	}

	for _, peering := range network.Peerings {
		if peering.Name == d.Id() {
			d.Set("name", peering.Name)
			d.Set("network", d.Get("network").(string))
			d.Set("auto_create_routes", peering.AutoCreateRoutes)
			d.Set("state", peering.State)
			d.Set("state_details", peering.StateDetails)
			return nil
		}
	}

	return handleNotFoundError(err, d, fmt.Sprintf("Network peering %q not found in network %q", d.Get("name").(string), networkName))

	return nil
}

func resourceComputeVpcPeeringConnectionDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	network, err := resourceComputeVpcPeeringConnectionNetwork(d)
	if err != nil {
		return err
	}

	peering := &compute.NetworksRemovePeeringRequest{
		Name: d.Id(),
	}

	// Delete the network peering
	op, err := config.clientCompute.Networks.RemovePeering(
		project, network, peering).Do()
	if err != nil {
		return fmt.Errorf("Error deleting network peering: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Deleting Network Peering")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
