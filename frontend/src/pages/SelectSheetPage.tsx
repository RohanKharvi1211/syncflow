import { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from 'react-query';
import { useAuthStore } from '@shared/store/authStore';
import { httpClient } from '@shared/adapters/httpClient';
import { dataObjectApi } from '@domains/pipelines/adapters/pipelineApi';

interface GoogleSheet {
  id: string;
  name: string;
  mimeType: string;
  modifiedTime: string;
  size?: string;
}

export function SelectSheetPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { setToken } = useAuthStore();
  const connectionId = searchParams.get('connection_id');
  const token = searchParams.get('token');
  const returnTo = searchParams.get('return_to'); // 'pipeline' or 'connections'
  const sourceAppId = searchParams.get('source_app_id'); // For pipeline creation flow

  const [selectedSheet, setSelectedSheet] = useState<GoogleSheet | null>(null);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (token) {
      setToken(token);
    }
  }, [token, setToken]);

  const { data: sheets, isLoading, error } = useQuery<GoogleSheet[]>(
    ['googleSheets', connectionId],
    () => httpClient.get<{ sheets: GoogleSheet[] }>(`/google-sheets/list?connection_id=${connectionId}`).then(res => res.sheets),
    {
      enabled: !!connectionId,
    }
  );

  const createDataObjectMutation = useMutation(
    (sheet: GoogleSheet) => dataObjectApi.createDataObject({
      connection_id: connectionId!,
      object_type: 'SHEET',
      identifier: sheet.id, // Store the Google Drive file ID
      config: {
        file_id: sheet.id,
        file_name: sheet.name,
        mime_type: sheet.mimeType,
        modified_time: sheet.modifiedTime,
        size: sheet.size,
      },
    }),
    {
      onSuccess: (dataObject) => {
        // Invalidate queries to refresh data
        queryClient.invalidateQueries(['connections']);
        queryClient.invalidateQueries(['dataObjects']);
        
        // If we came from pipeline creation, redirect back with connection and data object info
        if (returnTo === 'pipeline') {
          navigate(`/pipelines/create?source_connection_id=${connectionId}&source_data_object_id=${dataObject.id}`);
        } else {
          // Otherwise, go to connections page
          navigate('/connections');
        }
      },
      onError: (error: any) => {
        console.error('Failed to create data object:', error);
        alert(`Failed to save sheet: ${error.error || 'Unknown error'}`);
      },
    }
  );

  const handleSelectSheet = async (sheet: GoogleSheet) => {
    setSelectedSheet(sheet);
    // Create DataObject for this sheet
    // This stores the sheet details in the data_objects table
    createDataObjectMutation.mutate(sheet);
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading your Google Sheets...</p>
        </div>
      </div>
    );
  }

  if (error) {
    const errorMessage = (error as any)?.error || (error as any)?.message || 'Failed to load your Google Sheets. Please try again.';
    const errorDetails = (error as any)?.details || '';
    
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8">
          <div className="text-center">
            <div className="text-red-600 text-5xl mb-4">✕</div>
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Error Loading Sheets</h2>
            <p className="text-gray-600 mb-2">{errorMessage}</p>
            {errorDetails && (
              <p className="text-sm text-gray-500 mb-6">{errorDetails}</p>
            )}
            <div className="mt-4 space-y-2">
              <button
                onClick={() => window.location.reload()}
                className="w-full px-6 py-2 bg-gray-200 text-gray-800 rounded-lg hover:bg-gray-300"
              >
                Retry
              </button>
              <button
                onClick={() => navigate('/connections')}
                className="w-full px-6 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
              >
                Back to Connections
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-4xl mx-auto px-4">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Select a Google Sheet</h1>
          <p className="text-gray-600">Choose a Google Sheet to use as your data source</p>
        </div>

        {sheets && sheets.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {sheets.map((sheet) => (
              <div
                key={sheet.id}
                onClick={() => handleSelectSheet(sheet)}
                className={`p-6 bg-white rounded-lg shadow border-2 cursor-pointer transition-all ${
                  selectedSheet?.id === sheet.id
                    ? 'border-primary-500 bg-primary-50'
                    : 'border-gray-200 hover:border-gray-300 hover:shadow-md'
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <h3 className="text-lg font-semibold text-gray-900 mb-2">{sheet.name}</h3>
                    <p className="text-sm text-gray-500">
                      Modified: {new Date(sheet.modifiedTime).toLocaleDateString()}
                    </p>
                  </div>
                  <span className="text-3xl ml-4">📊</span>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="bg-white rounded-lg shadow p-8 text-center">
            <p className="text-gray-600 mb-4">No Google Sheets found in your account.</p>
            <button
              onClick={() => navigate('/connections')}
              className="px-6 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
            >
              Back to Connections
            </button>
          </div>
        )}

        {createDataObjectMutation.isLoading && (
          <div className="mt-8 bg-white rounded-lg shadow p-6 text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600 mx-auto mb-4"></div>
            <p className="text-gray-600">Saving sheet...</p>
          </div>
        )}
      </div>
    </div>
  );
}

