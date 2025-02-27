import { siteAdminAreaRoutes } from '../../site-admin/routes'
import { SiteAdminAreaRoute } from '../../site-admin/SiteAdminArea'
import { lazyComponent } from '../../util/lazyComponent'
import { SHOW_BUSINESS_FEATURES } from '../dotcom/productSubscriptions/features'

export const enterpriseSiteAdminAreaRoutes: readonly SiteAdminAreaRoute[] = [
    ...siteAdminAreaRoutes,
    {
        path: '/license',
        render: lazyComponent(
            () => import('./productSubscription/SiteAdminProductSubscriptionPage'),
            'SiteAdminProductSubscriptionPage'
        ),
        exact: true,
    },
    {
        path: '/dotcom/customers',
        render: lazyComponent(
            () => import('./dotcom/customers/SiteAdminCustomersPage'),
            'SiteAdminProductCustomersPage'
        ),
        condition: () => SHOW_BUSINESS_FEATURES,
        exact: true,
    },
    {
        path: '/dotcom/product/subscriptions/new',
        render: lazyComponent(
            () => import('./dotcom/productSubscriptions/SiteAdminCreateProductSubscriptionPage'),
            'SiteAdminCreateProductSubscriptionPage'
        ),
        condition: () => SHOW_BUSINESS_FEATURES,
        exact: true,
    },
    {
        path: '/dotcom/product/subscriptions/:subscriptionUUID',
        render: lazyComponent(
            () => import('./dotcom/productSubscriptions/SiteAdminProductSubscriptionPage'),
            'SiteAdminProductSubscriptionPage'
        ),
        condition: () => SHOW_BUSINESS_FEATURES,
        exact: true,
    },
    {
        path: '/dotcom/product/subscriptions',
        render: lazyComponent(
            () => import('./dotcom/productSubscriptions/SiteAdminProductSubscriptionsPage'),
            'SiteAdminProductSubscriptionsPage'
        ),
        condition: () => SHOW_BUSINESS_FEATURES,
        exact: true,
    },
    {
        path: '/dotcom/product/licenses',
        render: lazyComponent(
            () => import('./dotcom/productSubscriptions/SiteAdminProductLicensesPage'),
            'SiteAdminProductLicensesPage'
        ),
        condition: () => SHOW_BUSINESS_FEATURES,
        exact: true,
    },
    {
        path: '/auth/providers',
        render: lazyComponent(
            () => import('./SiteAdminAuthenticationProvidersPage'),
            'SiteAdminAuthenticationProvidersPage'
        ),
        exact: true,
    },
    {
        path: '/auth/external-accounts',
        render: lazyComponent(() => import('./SiteAdminExternalAccountsPage'), 'SiteAdminExternalAccountsPage'),
        exact: true,
    },
    {
        path: '/registry/extensions',
        render: lazyComponent(() => import('./SiteAdminRegistryExtensionsPage'), 'SiteAdminRegistryExtensionsPage'),
        exact: true,
    },

    {
        path: '/batch-changes',
        exact: true,
        render: lazyComponent(
            () => import('../batches/settings/BatchChangesSiteConfigSettingsArea'),
            'BatchChangesSiteConfigSettingsArea'
        ),
        condition: ({ batchChangesEnabled }) => batchChangesEnabled,
    },
    {
        path: '/batch-changes/executions',
        exact: true,
        render: lazyComponent(() => import('../batches/settings/BatchSpecExecutionsPage'), 'BatchSpecExecutionsPage'),
        condition: ({ batchChangesEnabled, batchChangesExecutionEnabled }) =>
            batchChangesEnabled && batchChangesExecutionEnabled,
    },

    // Code intelligence upload routes
    {
        path: '/code-intelligence/uploads',
        render: lazyComponent(() => import('../codeintel/list/CodeIntelUploadsPage'), 'CodeIntelUploadsPage'),
        exact: true,
    },
    {
        path: '/code-intelligence/uploads/:id',
        render: lazyComponent(() => import('../codeintel/detail/CodeIntelUploadPage'), 'CodeIntelUploadPage'),
        exact: true,
    },

    // Auto indexing routes
    {
        path: '/code-intelligence/indexes',
        render: lazyComponent(() => import('../codeintel/list/CodeIntelIndexesPage'), 'CodeIntelIndexesPage'),
        exact: true,
        condition: () => Boolean(window.context?.codeIntelAutoIndexingEnabled),
    },
    {
        path: '/code-intelligence/indexes/:id',
        render: lazyComponent(() => import('../codeintel/detail/CodeIntelIndexPage'), 'CodeIntelIndexPage'),
        exact: true,
        condition: () => Boolean(window.context?.codeIntelAutoIndexingEnabled),
    },

    // Code intelligence configuration
    {
        path: '/code-intelligence/configuration',
        render: lazyComponent(
            () => import('../codeintel/configuration/CodeIntelConfigurationPage'),
            'CodeIntelConfigurationPage'
        ),
        exact: true,
    },
    {
        path: '/code-intelligence/configuration/:id',
        render: lazyComponent(
            () => import('../codeintel/configuration/CodeIntelConfigurationPolicyPage'),
            'CodeIntelConfigurationPolicyPage'
        ),
        exact: true,
    },

    // Legacy routes
    {
        path: '/lsif-uploads/:id',
        render: lazyComponent(() => import('./SiteAdminLsifUploadPage'), 'SiteAdminLsifUploadPage'),
        exact: true,
    },
]
