import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useMutation, useQueryClient } from 'react-query';
import { useAuthStore } from '@shared/store/authStore';
import { dataObjectApi } from '@domains/pipelines/adapters/pipelineApi';
import { httpClient } from '@shared/adapters/httpClient';

interface QuickBooksObjectOption {
  id: string;
  name: string;
  description: string;
}

const QUICKBOOKS_OBJECTS: QuickBooksObjectOption[] = [
  {
    id: 'Customer',
    name: 'Customers',
    description: 'Download customer records from QuickBooks Online',
  },
  {
    id: 'Invoice',
    name: 'Invoices',
    description: 'Download invoices and related amounts from QuickBooks Online',
  },
  {
    id: 'Item',
    name: 'Items / Products',
    description: 'Download products and services from your QuickBooks catalog',
  },
];

export function SelectQuickBooksObjectPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { setToken } = useAuthStore();
  const queryClient = useQueryClient();

  const connectionId = searchParams.get('connection_id');
  const token = searchParams.get('token');
  const companyId = searchParams.get('company_id');
  const email = searchParams.get('email');
  const returnTo = searchParams.get('return_to'); // 'pipeline' or 'connections'
  // const sourceAppId = searchParams.get('source_app_id'); // For pipeline creation flow (not used yet but kept for parity)
  const { setUser } = useAuthStore();

  useEffect(() => {
    if (token) {
      // Set token and user immediately to prevent redirect to login
      setToken(token);
      
      // Set user from URL params if available, or fetch from backend
      if (companyId) {
        setUser({
          id: '', // Will be fetched from token if needed
          company_id: companyId,
          email: email ? decodeURIComponent(email) : '',
          role: 'user' as const,
          is_active: true,
        });
      } else {
        // Try to fetch current user from backend
        httpClient.get('/users/me')
          .then((response: any) => {
            if (response.user) {
              setUser(response.user);
            }
          })
          .catch((error) => {
            console.error('Failed to fetch user:', error);
          });
      }
    } else if (!token && !connectionId) {
      // If no token and no connection, redirect to login
      navigate('/login');
    }
  }, [token, companyId, email, connectionId, setToken, setUser, navigate]);

  const createDataObjectMutation = useMutation(
    (qbObject: QuickBooksObjectOption) =>
      dataObjectApi.createDataObject({
        connection_id: connectionId!,
        // Treat QuickBooks entities as generic OBJECTs; identifier carries the specific type
        object_type: 'OBJECT',
        identifier: qbObject.id, // e.g. "Customer", "Invoice", "Item"
        config: {
          object_name: qbObject.name,
          object_id: qbObject.id,
          source: 'quickbooks',
        },
      }),
    {
      onSuccess: (dataObject) => {
        // Refresh connection/data object lists
        queryClient.invalidateQueries(['connections']);
        queryClient.invalidateQueries(['dataObjects']);

        if (returnTo === 'pipeline') {
          // Go back to pipeline creation with pre-selected source
          navigate(
            `/pipelines/create?source_connection_id=${connectionId}&source_data_object_id=${dataObject.id}`
          );
        } else {
          // Default: back to connections list
          navigate('/connections');
        }
      },
      onError: (error: any) => {
        console.error('Failed to create QuickBooks data object:', error);
        const message =
          error?.response?.data?.error ||
          error?.error ||
          error?.message ||
          'Unknown error';
        alert(`Failed to save QuickBooks object: ${message}`);
      },
    }
  );

  const handleSelectObject = (obj: QuickBooksObjectOption) => {
    if (!connectionId) {
      alert('Missing QuickBooks connection. Please reconnect and try again.');
      return;
    }
    createDataObjectMutation.mutate(obj);
  };

  if (!connectionId) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8 text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">
            QuickBooks Connection Not Found
          </h2>
          <p className="text-gray-600 mb-6">
            We couldn&apos;t find a QuickBooks connection in this link.
          </p>
          <button
            onClick={() => navigate('/connections')}
            className="px-6 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
          >
            Back to Connections
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-4xl mx-auto px-4">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">
            Select QuickBooks Object
          </h1>
          <p className="text-gray-600">
            Choose which QuickBooks data you want to download and use in your
            pipelines.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {QUICKBOOKS_OBJECTS.map((obj) => (
            <button
              key={obj.id}
              onClick={() => handleSelectObject(obj)}
              className="text-left p-6 bg-white rounded-lg shadow border border-gray-200 hover:border-primary-500 hover:shadow-md transition-all focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2"
            >
              <h3 className="text-lg font-semibold text-gray-900 mb-2">
                {obj.name}
              </h3>
              <p className="text-sm text-gray-600 mb-3">{obj.description}</p>
              <span className="inline-flex items-center text-sm font-medium text-primary-600">
                Use this object
                <span className="ml-1">{'>'}</span>
              </span>
            </button>
          ))}
        </div>

        {createDataObjectMutation.isLoading && (
          <div className="mt-8 bg-white rounded-lg shadow p-6 text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600 mx-auto mb-4"></div>
            <p className="text-gray-600">Saving QuickBooks object...</p>
          </div>
        )}
      </div>
    </div>
  );
}


