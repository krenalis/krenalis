//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package stripe

import (
	"github.com/meergo/meergo/core/types"
)

// https://docs.stripe.com/api/customers/create and
// https://docs.stripe.com/api/customers/update.
//
// The "source" and "test_clock" fields has been excluded because it is not relevant to Meergo.

var destinationSchema = types.Object([]types.Property{
	{
		Name:     "address",
		Type:     destinationAddress,
		Nullable: true,
	},
	{
		Name:     "description",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "email",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name: "metadata",
		Type: types.Map(types.Text()),
	},
	{
		Name:     "name",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		// Only when creating.
		Name:        "payment_method",
		Type:        types.Text(),
		Nullable:    true,
		Description: "Used only when creating a customer, ignored when updating.",
	},
	{
		Name:     "phone",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "shipping",
		Type:     destinationShipping,
		Nullable: true,
	},
	{
		Name: "tax",
		Type: destinationTax,
	},
	{
		Name: "balance",
		Type: types.Int(64).WithIntRange(-1000000000000, 1000000000000),
	},
	{
		Name: "cash_balance",
		Type: types.Object([]types.Property{
			{
				Name: "settings",
				Type: types.Object([]types.Property{
					{
						Name:     "reconciliation_mode",
						Type:     types.Text().WithValues("automatic", "manual", "merchant_default"),
						Nullable: true,
					},
				}),
				Nullable: true,
			},
		}),
	},
	{
		// Only when updating.
		Name:        "default_source",
		Type:        types.Text(),
		Description: "Used only when updating a customer, ignored when creating.",
	},
	{
		Name: "invoice_prefix",
		Type: types.Text().WithByteLen(12),
	},
	{
		Name: "invoice_settings",
		Type: destinationInvoiceSettings,
	},
	{
		// If the Stripe account applies account-level sequencing, this
		// parameter is ignored in API requests and excluded from API responses.
		Name: "next_invoice_sequence",
		Type: types.Int(32).WithIntRange(1, 1000000000),
	},
	{
		Name: "preferred_locales",
		Type: types.Array(types.Text()),
	},
	{
		Name:     "tax_exempt",
		Type:     types.Text().WithValues("none", "exempt", "reverse"),
		Nullable: true,
	},
	{
		// Only when creating.
		Name: "tax_id_data",
		Type: types.Array(types.Object([]types.Property{
			{
				Name:           "type",
				Type:           types.Text().WithValues("ad_nrt", "ae_trn", "al_tin", "am_tin", "ao_tin", "ar_cuit", "au_abn", "au_arn", "aw_tin", "az_tin", "ba_tin", "bb_tin", "bd_bin", "bf_ifu", "bg_uic", "bh_vat", "bj_ifu", "bo_tin", "br_cnpj", "br_cpf", "bs_tin", "by_tin", "ca_bn", "ca_gst_hst", "ca_pst_bc", "ca_pst_mb", "ca_pst_sk", "ca_qst", "cd_nif", "ch_uid", "ch_vat", "cl_tin", "cm_niu", "cn_tin", "co_nit", "cr_tin", "cv_nif", "de_stn", "do_rcn", "ec_ruc", "eg_tin", "es_cif", "et_tin", "eu_oss_vat", "eu_vat", "gb_vat", "ge_vat", "gn_nif", "hk_br", "hr_oib", "hu_tin", "id_npwp", "il_vat", "in_gst", "is_vat", "jp_cn", "jp_rn", "jp_trn", "ke_pin", "kg_tin", "kh_tin", "kr_brn", "kz_bin", "la_tin", "li_uid", "li_vat", "ma_vat", "md_vat", "me_pib", "mk_vat", "mr_nif", "mx_rfc", "my_frp", "my_itn", "my_sst", "ng_tin", "no_vat", "no_voec", "np_pan", "nz_gst", "om_vat", "pe_ruc", "ph_tin", "ro_tin", "rs_pib", "ru_inn", "ru_kpp", "sa_vat", "sg_gst", "sg_uen", "si_tin", "sn_ninea", "sr_fin", "sv_nit", "th_vat", "tj_tin", "tr_tin", "tw_vat", "tz_vat", "ua_vat", "ug_tin", "us_ein", "uy_ruc", "uz_tin", "uz_vat", "ve_rif", "vn_tin", "za_vat", "zm_tin", "zw_tin"),
				CreateRequired: true,
			},
			{
				Name:           "value",
				Type:           types.Text(),
				CreateRequired: true,
			},
		})),
		Description: "Used only when creating a customer, ignored when updating.",
	},
})

var destinationAddress = types.Object([]types.Property{
	{
		Name:     "city",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "country",
		Type:     types.Text(), // don't limit to 2 chars: ISO 3166-1 alpha-2 is recommended but not enforced by Stripe.
		Nullable: true,
	},
	{
		Name:     "line1",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "line2",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "postal_code",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "state",
		Type:     types.Text(),
		Nullable: true,
	},
})

var destinationShipping = types.Object([]types.Property{
	{
		Name:           "address",
		Type:           destinationAddress,
		CreateRequired: true,
		UpdateRequired: true,
	},
	{
		Name:           "name",
		Type:           types.Text(),
		CreateRequired: true,
		UpdateRequired: true,
	},
	{
		Name:     "phone",
		Type:     types.Text(),
		Nullable: true,
	},
})

var destinationTax = types.Object([]types.Property{
	{
		Name: "ip_address",
		Type: types.Text(),
	},
	{
		Name: "validate_location",
		Type: types.Text().WithValues("auto", "deferred", "immediately"), // "auto" is only allowed during update
	},
})

var destinationInvoiceSettings = types.Object([]types.Property{
	{
		Name: "custom_fields",
		Type: types.Array(types.Object([]types.Property{
			{
				Name:           "name",
				Type:           types.Text().WithCharLen(40),
				Nullable:       true,
				CreateRequired: true,
				UpdateRequired: true,
			},
			{
				Name:           "value",
				Type:           types.Text().WithByteLen(140),
				Nullable:       true,
				CreateRequired: true,
				UpdateRequired: true,
			},
		})).WithMaxElements(4),
		Nullable: true,
	},
	{
		Name:     "default_payment_method",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "footer",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name: "rendering_options",
		Type: types.Object([]types.Property{
			{
				Name:     "amount_tax_display",
				Type:     types.Text().WithValues("exclude_tax", "include_inclusive_tax"),
				Nullable: true,
			},
			{
				Name:     "template",
				Type:     types.Text(),
				Nullable: true,
			},
		}),
		Nullable: true,
	},
})
