Installing KRIB (Kubernetes Rebar Integrated Bootstrapping)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

This is about installing KRIB on an existing DRP endpoint.  See :ref:`component_krib` for instructions on using and extending KRIB.

.. _rs_krib:

Prerequists
-----------

You need to install the system with a `discovery` workflow that includes `sledgehammer` as a minimum.  These steps are documented in :ref:`rs_quickstart`.

Install the DRP command line interface (`drpcli`) as per :ref:`rs_cli`

You also need to install some common utilities: git, curl and jq.

Setup DRP Access
----------------

We need to make sure we have access to the system via the CLI.

  ::

    # UPDATE THESE: RS_ENDPOINT is not needed if using localhost
    export RS_ENDPOINT="[endpoint URL]"
    # UPDATE THESE: RS_KEY is not needed if using defaults
    export RS_KEY="[endpoint user:password]"
    # verify credentials
    drpcli get info

Setup Packet API Integration
----------------------------

If you are using Packet.net as a reference platform.  The following steps are specific to that platform.

Make sure you set your information in the exports!

  ::

    # UPDATE THESE: Packet Project Information for Plugin
    export PACKET_API_KEY="[packet_api_key]"
    export PACKET_PROJECT_ID="[packet_project_id]"
    # download plugin provider (update for version or archtiecture)
    drpcli plugin_providers upload packet-ipmi from catalog:packet-ipmi-tip
    drpcli plugins create '{ "Name": "packet-ipmi",
       "Params": {
         "packet/api-key": "$PACKET_API_KEY",
         "packet/project-id": "$PACKET_PROJECT_ID"
       },
       "Provider": "packet-ipmi"
      }'
    # verify it worked - should return true
    drpcli plugins show packet-ipmi | jq .Available

.. note:: The URLs provided for plugin downloads will change overtime for newer versions


Install KRIB Components and Prerequisites
-----------------------------------------

The following steps will install the required plugins and content for KRIB to the tip version.  Change `tip` to `stable' to use the stable version.

  ::

    export KRIBVER="tip"
    drpcli plugin_providers upload certs from catalog:certs-$KRIBVER
    drpcli contents upload catalog:drp-community-content-$KRIBVER
    drpcli contents upload catalog:task-library-$KRIBVER
    drpcli contents upload catalog:krib-$KRIBVER

.. note:: This is maintained with more detail at :ref:`component_krib`.

Create Machines
---------------

To create machines in Packet without a Terraform, use the following command:

  ::

    # copy multiple times to create extra machines - names must be unique
    drpcli machines create '{ "Name": "krib-0",
       "Params": {
         "machine-plugin": "packet-ipmi"
       }
    }'

Running KRIB
------------

Continue to next steps on :ref:`component_krib`.