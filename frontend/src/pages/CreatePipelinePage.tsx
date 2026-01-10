import React, { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from 'react-query';
import { useAuthStore } from '@shared/store/authStore';
import { useCompanyStore } from '@shared/store/companyStore';
import { connectionApi } from '@domains/connections/adapters/connectionApi';
import { dataObjectApi } from '@domains/pipelines/adapters/pipelineApi';
import { pipelineApi } from '@domains/pipelines/adapters/pipelineApi';
import { appApi } from '@domains/apps/adapters/appApi';
import { httpClient } from '@shared/adapters/httpClient';
import { Connection } from '@domains/connections/entities/Connection';
import { DataObject } from '@domains/pipelines/entities/Pipeline';
import { App } from '@domains/apps/entities/App';
import { QuickBooksCredentialsModal } from '@shared/components/QuickBooksCredentialsModal';

type Step = 1 | 2 | 3;

interface FieldMapping {
  source: string;
  destination: string;
}

function DestinationObjectSelect({
  connectionId,
  value,
  onChange,
}: {
  connectionId: string;
  value: string;
  onChange: (obj: DataObject | null) => void;
}) {
  const { data: objects } = useQuery<DataObject[]>(
    ['dataObjects', connectionId],
    () => dataObjectApi.getDataObjects(connectionId),
    { enabled: !!connectionId }
  );

  return (
    <select
      value={value}
      onChange={(e) => {
        const obj = objects?.find((o) => o.id === e.target.value);
        onChange(obj || null);
      }}
      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-primary-500 focus:border-primary-500"
    >
      <option value="">Select Salesforce Object</option>
      {objects?.map((obj) => (
        <option key={obj.id} value={obj.id}>
          {obj.identifier}
        </option>
      ))}
    </select>
  );
}

export function CreatePipelinePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const { selectedCompany } = useCompanyStore();

  const [step, setStep] = useState<Step>(1);
  const [sourceConnection, setSourceConnection] = useState<Connection | null>(null);
  const [destinationConnection, setDestinationConnection] = useState<Connection | null>(null);
  const [sourceDataObject, setSourceDataObject] = useState<DataObject | null>(null);
  const [destinationDataObject, setDestinationDataObject] = useState<DataObject | null>(null);
  const [fieldMappings, setFieldMappings] = useState<FieldMapping[]>([]);
  const [showQuickBooksModal, setShowQuickBooksModal] = useState(false);
  const [pendingQuickBooksApp, setPendingQuickBooksApp] = useState<App | null>(null);
  const [pendingIsSource, setPendingIsSource] = useState(false);

  const companyId = selectedCompany?.id || user?.company_id || '';

  // Fetch connections first (before useEffect that depends on it)
  const { data: connections } = useQuery<Connection[]>(
    ['connections', companyId],
    () => connectionApi.getConnections(companyId),
    { enabled: !!companyId }
  );

  // Check if returning from sheet selection - handle both source and destination
  useEffect(() => {
    const sourceConnectionId = searchParams.get('source_connection_id');
    const sourceDataObjectId = searchParams.get('source_data_object_id');
    const destinationConnectionId = searchParams.get('destination_connection_id');
    const destinationDataObjectId = searchParams.get('destination_data_object_id');
    
    // Restore source connection and data object
    if (sourceConnectionId && connections) {
      const conn = connections.find(c => c.id === sourceConnectionId);
      if (conn) {
        setSourceConnection(conn);
        
        if (sourceDataObjectId) {
          dataObjectApi.getDataObject(sourceDataObjectId).then(obj => {
            setSourceDataObject(obj);
          }).catch(err => {
            console.error('Failed to load source data object:', err);
          });
        }
      }
    }
    
    // Restore destination connection and data object
    if (destinationConnectionId && connections) {
      const conn = connections.find(c => c.id === destinationConnectionId);
      if (conn) {
        setDestinationConnection(conn);
        
        if (destinationDataObjectId) {
          dataObjectApi.getDataObject(destinationDataObjectId).then(obj => {
            setDestinationDataObject(obj);
          }).catch(err => {
            console.error('Failed to load destination data object:', err);
          });
        }
      }
    }
    
    // Determine step based on what's selected
    if (sourceConnectionId && sourceDataObjectId) {
      if (destinationConnectionId && destinationDataObjectId) {
        setStep(3); // Both source and destination selected - move to mapping step
      } else {
        setStep(2); // Only source selected - move to destination selection step
      }
    } else if (destinationConnectionId && destinationDataObjectId) {
      // Only destination selected (shouldn't happen normally, but handle it)
      setStep(2);
    }
  }, [searchParams, connections]);

  // Fetch apps for source and destination
  const { data: sourceApps } = useQuery<App[]>(
    ['apps', 'source'],
    () => appApi.getApps('source')
  );

  const { data: destinationApps } = useQuery<App[]>(
    ['apps', 'destination'],
    () => appApi.getApps('destination')
  );

  const handleCreateConnection = async (app: App, isSource: boolean) => {
    // For Google apps, initiate OAuth
    if (app.name === 'googlesheet' || app.name === 'googledrive') {
      try {
        // Always use 'pipeline' as return_to when coming from pipeline creation
        // Include is_source in the URL to distinguish between source and destination
        const isSourceParam = isSource ? 'true' : 'false';
        // Store current selections in sessionStorage to preserve them after OAuth
        sessionStorage.setItem('pipeline_source_connection_id', sourceConnection?.id || '');
        sessionStorage.setItem('pipeline_source_data_object_id', sourceDataObject?.id || '');
        sessionStorage.setItem('pipeline_destination_connection_id', destinationConnection?.id || '');
        sessionStorage.setItem('pipeline_destination_data_object_id', destinationDataObject?.id || '');
        sessionStorage.setItem('pipeline_step', step.toString());
        sessionStorage.setItem('pipeline_is_source', isSourceParam);
        
        const response = await fetch(`http://localhost:8080/api/oauth/google/initiate?user_id=temp&return_to=pipeline&app_id=${app.id}&is_source=${isSourceParam}`);
        const data = await response.json();
        if (data.auth_url) {
          window.location.href = data.auth_url;
        }
      } catch (error) {
        console.error('Failed to initiate OAuth:', error);
      }
    } else if (app.name === 'quickbooks') {
      // For QuickBooks, show credentials modal first
      setPendingQuickBooksApp(app);
      setPendingIsSource(isSource);
      setShowQuickBooksModal(true);
    } else {
      alert(`Connection setup for ${app.display_name} is not yet implemented`);
    }
  };

  const { data: sourceDataObjects } = useQuery<DataObject[]>(
    ['dataObjects', sourceConnection?.id],
    () => dataObjectApi.getDataObjects(sourceConnection?.id || ''),
    { enabled: !!sourceConnection }
  );

  // Auto-advance to step 3 when both source and destination are selected
  useEffect(() => {
    if (sourceDataObject && destinationDataObject && step === 2) {
      setStep(3);
    }
  }, [sourceDataObject, destinationDataObject, step]);

  // Fetch fields for source and destination data objects
  // Using new generic /data-objects/:id/fields endpoint that works for both Google Sheets and QuickBooks
  const { data: sourceFields, error: sourceFieldsError, refetch: refetchSourceFields, isFetching: isFetchingSourceFields } = useQuery<string[]>(
    ['dataObjectFields', 'source', sourceDataObject?.id],
    () => {
      if (!sourceDataObject?.id) return Promise.resolve([]);
      return httpClient.get<{ fields: string[] }>(`/data-objects/${sourceDataObject.id}/fields`)
        .then(res => res.fields)
        .catch((error: any) => {
          // Re-throw to let React Query handle it
          throw error;
        });
    },
    { enabled: !!sourceDataObject?.id && step === 3, retry: false }
  );

  const { data: destinationFields, error: destinationFieldsError, refetch: refetchDestinationFields, isFetching: isFetchingDestinationFields } = useQuery<string[]>(
    ['dataObjectFields', 'destination', destinationDataObject?.id],
    () => {
      if (!destinationDataObject?.id) return Promise.resolve([]);
      return httpClient.get<{ fields: string[] }>(`/data-objects/${destinationDataObject.id}/fields`)
        .then(res => res.fields)
        .catch((error: any) => {
          // Re-throw to let React Query handle it
          throw error;
        });
    },
    { enabled: !!destinationDataObject?.id && step === 3, retry: false }
  );

  // Retry function to refetch both fields
  const handleRetry = () => {
    if (sourceDataObject?.id) {
      refetchSourceFields();
    }
    if (destinationDataObject?.id) {
      refetchDestinationFields();
    }
  };

  // Initialize field mappings when fields are loaded
  // Mapping direction: destination → source (which destination field uses which source field)
  useEffect(() => {
    // Only initialize when we reach step 3 and both source and destination fields are loaded
    // Only initialize once when fields first become available (when fieldMappings is empty)
    if (step === 3 && destinationFields && destinationFields.length > 0 && sourceFields && sourceFields.length > 0 && fieldMappings.length === 0) {
      // Initialize with empty mappings for all destination fields
      // Each mapping represents: destination field → source field (what source field should fill this destination field)
      const initialMappings = destinationFields.map(destField => ({
        destination: destField, // Destination field (e.g., Google Sheet column header)
        source: '' // Source field (e.g., QuickBooks Invoice field) - user will select
      }));
      setFieldMappings(initialMappings);
    }
  }, [step, destinationFields, sourceFields, fieldMappings.length]);

  const createPipelineMutation = useMutation(
    (data: any) => pipelineApi.createPipeline(data),
    {
      onSuccess: () => {
        // Invalidate all pipeline queries to ensure the list refreshes
        queryClient.invalidateQueries({ queryKey: ['pipelines'] });
        navigate('/pipelines');
      },
    }
  );

  const handleNext = () => {
    if (step === 1 && sourceDataObject) {
      // Move to step 2 (destination setup)
      setStep(2);
    } else if (step === 2 && sourceDataObject && destinationDataObject) {
      // Move to step 3 (data mapping)
      setStep(3);
    } else if (step === 3) {
      // Create pipeline
      handleSubmit();
    }
  };

  const handleSubmit = () => {
    if (!sourceDataObject || !destinationDataObject) return;

    const mappingObj: Record<string, any> = {};
    fieldMappings.forEach((mapping) => {
      if (mapping.source && mapping.destination) {
        mappingObj[mapping.destination] = {
          type: 'direct',
          source_field: mapping.source,
        };
      }
    });

    createPipelineMutation.mutate({
      company_id: companyId,
      source_object_id: sourceDataObject.id,
      destination_object_id: destinationDataObject.id,
      sync_type: 'PULL',
      schedule_interval: 60,
      field_mapping: mappingObj,
    });
  };

  const addFieldMapping = () => {
    setFieldMappings([...fieldMappings, { source: '', destination: '' }]);
  };

  const updateFieldMapping = (index: number, field: 'source' | 'destination', value: string) => {
    const updated = [...fieldMappings];
    updated[index][field] = value;
    setFieldMappings(updated);
  };

  const removeFieldMapping = (index: number) => {
    setFieldMappings(fieldMappings.filter((_, i) => i !== index));
  };

  // Helper function to render error message
  const renderError = (error: unknown): React.ReactNode => {
    const errorData = (error as any) || {};
    
    const isPermissionError = 
      errorData.error === 'Google Sheets API Permission Required' ||
      errorData.error === 'Google Sheets API is not enabled' ||
      (errorData.details && typeof errorData.details === 'string' && (
        errorData.details.includes('API has not been used') ||
        errorData.details.includes('it is disabled') ||
        errorData.details.includes('PERMISSION_DENIED')
      ));
    
    if (isPermissionError) {
      return (
        <div className="text-center space-y-4">
          <div>
            <p className="text-lg font-semibold text-red-900 mb-2">
              ⚠️ {String(errorData.error || 'Permission Error')}
            </p>
            <p className="text-sm text-red-800 mb-4">
              {String(errorData.message || 'The Google Sheets API needs to be enabled in your Google Cloud project.')}
            </p>
          </div>
          
          {errorData.enable_url ? (
            <a 
              href={String(errorData.enable_url)} 
              target="_blank" 
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-medium transition-colors shadow-md"
            >
              🔗 Enable Google Sheets API
            </a>
          ) : errorData.help_url ? (
            <a 
              href={String(errorData.help_url)} 
              target="_blank" 
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-medium transition-colors shadow-md"
            >
              🔗 Open Google Cloud Console
            </a>
          ) : (
            <a 
              href="https://console.developers.google.com/apis/api/sheets.googleapis.com/overview" 
              target="_blank" 
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-medium transition-colors shadow-md"
            >
              🔗 Enable Google Sheets API
            </a>
          )}
          
          <div className="flex flex-col sm:flex-row gap-3 justify-center mt-4">
            <button
              onClick={handleRetry}
              disabled={isFetchingSourceFields || isFetchingDestinationFields}
              className="px-6 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 font-medium transition-colors shadow-md disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isFetchingSourceFields || isFetchingDestinationFields ? 'Retrying...' : '🔄 Retry'}
            </button>
          </div>
        </div>
      );
    }
    
    return (
      <div className="text-sm text-red-800">
        <p className="font-medium mb-1">{String(errorData.error || 'Failed to load fields')}</p>
        {errorData.details && (
          <p className="text-xs text-red-600 mt-1">{String(errorData.details)}</p>
        )}
      </div>
    );
  };

  return (
    <div className="max-w-7xl mx-auto">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900">Create New Connection</h1>
        <div className="flex items-center mt-4 space-x-4">
          <div className={`flex items-center ${step >= 1 ? 'text-primary-600' : 'text-gray-400'}`}>
            <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
              step >= 1 ? 'bg-primary-600 text-white' : 'bg-gray-200 text-gray-500'
            }`}>
              1
            </div>
            <span className="ml-2 font-medium">Source Setup</span>
          </div>
          <div className="w-12 h-0.5 bg-gray-300"></div>
          <div className={`flex items-center ${step >= 2 ? 'text-primary-600' : 'text-gray-400'}`}>
            <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
              step >= 2 ? 'bg-primary-600 text-white' : 'bg-gray-200 text-gray-500'
            }`}>
              2
            </div>
            <span className="ml-2 font-medium">Destination Setup</span>
          </div>
          <div className="w-12 h-0.5 bg-gray-300"></div>
          <div className={`flex items-center ${step >= 3 ? 'text-primary-600' : 'text-gray-400'}`}>
            <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
              step >= 3 ? 'bg-primary-600 text-white' : 'bg-gray-200 text-gray-500'
            }`}>
              3
            </div>
            <span className="ml-2 font-medium">Data Mapping</span>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow-lg p-8">
        <div className="grid grid-cols-2 gap-8">
          {/* Left Panel - Source Setup */}
          <div className="border-r pr-8">
            <h2 className="text-xl font-semibold text-gray-900 mb-6">Source Setup</h2>
            
            <div className="space-y-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Select Source
                </label>
                <div className="flex items-center space-x-2">
                  <select
                    value={sourceConnection?.id || ''}
                    onChange={(e) => {
                      if (e.target.value.startsWith('app_')) {
                        // User selected an app that needs connection
                        const appId = e.target.value.replace('app_', '');
                        const app = sourceApps?.find((a) => a.id === appId);
                        if (app) {
                          handleCreateConnection(app, true); // true = isSource
                        }
                        return;
                      }
                      const conn = connections?.find((c) => c.id === e.target.value);
                      setSourceConnection(conn || null);
                      // If connection has a data object, auto-select it
                      if (conn && conn.data_objects && conn.data_objects.length > 0) {
                        setSourceDataObject(conn.data_objects[0] as any);
                      } else {
                        setSourceDataObject(null);
                      }
                    }}
                    className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-primary-500 focus:border-primary-500"
                  >
                    <option value="">Select source...</option>
                    {/* Show all connections - they can be used for both source and destination */}
                    {connections && connections.length > 0 ? (
                      connections.map((conn) => {
                        // Show sheet name for Google Sheets/Drive connections
                        let displayName = conn.app?.display_name || conn.type || 'Unknown';
                        if ((conn.app?.name === 'googlesheet' || conn.app?.name === 'googledrive') && conn.data_objects && conn.data_objects.length > 0) {
                          const dataObject = conn.data_objects[0];
                          let config = dataObject.config;
                          if (typeof config === 'string') {
                            try {
                              config = JSON.parse(config);
                            } catch (e) {
                              // Ignore parse errors
                            }
                          }
                          const sheetName = config?.file_name || config?.fileName || dataObject.identifier;
                          displayName = `Google Sheet - ${sheetName}`;
                        }
                        return (
                          <option key={conn.id} value={conn.id}>
                            {displayName}
                          </option>
                        );
                      })
                    ) : null}
                    {/* Show apps as options to create new connections */}
                    {sourceApps && sourceApps.length > 0 && (
                      <optgroup label="Available Apps (Create Connection)">
                        {sourceApps.map((app) => (
                          <option key={app.id} value={`app_${app.id}`}>
                            {app.display_name} (Click to connect)
                          </option>
                        ))}
                      </optgroup>
                    )}
                  </select>
                  {sourceConnection?.app?.name === 'googlesheet' && (
                    <span className="text-2xl">📊</span>
                  )}
                </div>
                {/* Show sheet selection prompt only if connection has no data objects */}
                {sourceConnection && (sourceConnection.app?.name === 'googlesheet' || sourceConnection.app?.name === 'googledrive') && 
                 (!sourceConnection.data_objects || sourceConnection.data_objects.length === 0) && !sourceDataObject && (
                  <div className="mt-4 p-4 bg-blue-50 border border-blue-200 rounded-lg">
                    <p className="text-sm text-blue-800 mb-2">
                      Please select a Google Sheet to continue
                    </p>
                    <button
                      onClick={() => {
                        navigate(`/connections/select-sheet?connection_id=${sourceConnection.id}&return_to=pipeline&source_app_id=${sourceConnection.app?.id}`);
                      }}
                      className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
                    >
                      Select Sheet
                    </button>
                  </div>
                )}
                
                {/* Show selected sheet info if data object exists */}
                {sourceConnection && sourceDataObject && (
                  <div className="mt-4 p-4 bg-green-50 border border-green-200 rounded-lg">
                    <p className="text-sm text-green-800">
                      ✓ Selected: {(() => {
                        let config = sourceDataObject.config;
                        if (typeof config === 'string') {
                          try {
                            config = JSON.parse(config);
                          } catch (e) {
                            // Ignore parse errors
                          }
                        }
                        return config?.file_name || config?.fileName || sourceDataObject.identifier;
                      })()}
                    </p>
                  </div>
                )}
                
                {/* Show data object selection for other connection types */}
                {sourceConnection && sourceConnection.app?.name !== 'googlesheet' && sourceConnection.app?.name !== 'googledrive' && (
                  <div className="mt-4">
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Select Data Object
                    </label>
                    <select
                      value={sourceDataObject?.id || ''}
                      onChange={(e) => {
                        const obj = sourceDataObjects?.find((o) => o.id === e.target.value);
                        setSourceDataObject(obj || null);
                      }}
                      className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-primary-500 focus:border-primary-500"
                    >
                      <option value="">Select data object...</option>
                      {sourceDataObjects?.map((obj) => (
                        <option key={obj.id} value={obj.id}>
                          {obj.identifier}
                        </option>
                      ))}
                    </select>
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Right Panel - Destination Setup */}
          <div className="pl-8">
            <h2 className="text-xl font-semibold text-gray-900 mb-6">Destination Setup</h2>
            
            <div className="space-y-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Select Destination
                </label>
                <div className="flex items-center space-x-2">
                  <select
                    value={destinationConnection?.id || ''}
                    onChange={(e) => {
                      if (e.target.value.startsWith('app_')) {
                        // User selected an app that needs connection
                        const appId = e.target.value.replace('app_', '');
                        const app = destinationApps?.find((a) => a.id === appId);
                        if (app) {
                          handleCreateConnection(app, false); // false = isDestination
                        }
                        return;
                      }
                      const conn = connections?.find((c) => c.id === e.target.value);
                      setDestinationConnection(conn || null);
                      // If connection has a data object, auto-select it
                      if (conn && conn.data_objects && conn.data_objects.length > 0) {
                        setDestinationDataObject(conn.data_objects[0] as any);
                      } else {
                        setDestinationDataObject(null);
                      }
                    }}
                    className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-primary-500 focus:border-primary-500"
                  >
                    <option value="">Select destination...</option>
                    {/* Show all connections - they can be used for both source and destination */}
                    {connections && connections.length > 0 ? (
                      connections.map((conn) => {
                        // Show sheet name for Google Sheets/Drive connections
                        let displayName = conn.app?.display_name || conn.type || 'Unknown';
                        if ((conn.app?.name === 'googlesheet' || conn.app?.name === 'googledrive') && conn.data_objects && conn.data_objects.length > 0) {
                          const dataObject = conn.data_objects[0];
                          let config = dataObject.config;
                          if (typeof config === 'string') {
                            try {
                              config = JSON.parse(config);
                            } catch (e) {
                              // Ignore parse errors
                            }
                          }
                          const sheetName = config?.file_name || config?.fileName || dataObject.identifier;
                          displayName = `Google Sheet - ${sheetName}`;
                        }
                        return (
                          <option key={conn.id} value={conn.id}>
                            {displayName}
                          </option>
                        );
                      })
                    ) : null}
                    {/* Show apps as options to create new connections */}
                    {destinationApps && destinationApps.length > 0 && (
                      <optgroup label="Available Apps (Create Connection)">
                        {destinationApps.map((app) => (
                          <option key={app.id} value={`app_${app.id}`}>
                            {app.display_name} (Click to connect)
                          </option>
                        ))}
                      </optgroup>
                    )}
                  </select>
                  {destinationConnection?.app?.name === 'salesforce' && (
                    <span className="text-2xl">☁️</span>
                  )}
                </div>
              </div>

              {destinationConnection && (
                <>
                  {/* Show sheet selection prompt only if connection has no data objects */}
                  {destinationConnection && (destinationConnection.app?.name === 'googlesheet' || destinationConnection.app?.name === 'googledrive') && 
                   (!destinationConnection.data_objects || destinationConnection.data_objects.length === 0) && !destinationDataObject && (
                    <div className="mt-4 p-4 bg-blue-50 border border-blue-200 rounded-lg">
                      <p className="text-sm text-blue-800 mb-2">
                        Please select a Google Sheet to continue
                      </p>
                      <button
                        onClick={() => {
                          navigate(`/connections/select-sheet?connection_id=${destinationConnection.id}&return_to=pipeline&source_app_id=${destinationConnection.app?.id}`);
                        }}
                        className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
                      >
                        Select Sheet
                      </button>
                    </div>
                  )}
                  
                  {/* Show selected sheet info if data object exists */}
                  {destinationConnection && destinationDataObject && (
                    <div className="mt-4 p-4 bg-green-50 border border-green-200 rounded-lg">
                      <p className="text-sm text-green-800">
                        ✓ Selected: {(() => {
                          let config = destinationDataObject.config;
                          if (typeof config === 'string') {
                            try {
                              config = JSON.parse(config);
                            } catch (e) {
                              // Ignore parse errors
                            }
                          }
                          return config?.file_name || config?.fileName || destinationDataObject.identifier;
                        })()}
                      </p>
                    </div>
                  )}
                  
                  {/* Show data object selection for non-Google Sheets connections */}
                  {destinationConnection && destinationConnection.app?.name !== 'googlesheet' && destinationConnection.app?.name !== 'googledrive' && (
                    <div className="mt-4">
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Target Object
                      </label>
                      <DestinationObjectSelect
                        connectionId={destinationConnection.id}
                        value={destinationDataObject?.id || ''}
                        onChange={setDestinationDataObject}
                      />
                    </div>
                  )}

                </>
              )}
            </div>
          </div>
        </div>

        {/* Step 3: Data Mapping - Full Width */}
        {step === 3 && (
          <div className="mt-8 pt-8 border-t">
            <h2 className="text-xl font-semibold text-gray-900 mb-6">Data Mapping</h2>
            {sourceFields && destinationFields ? (
              <div className="space-y-3">
                <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
                  <p className="text-sm text-blue-800">
                    <strong>Mapping Direction:</strong> For each destination field, select which source field should populate it.
                  </p>
                </div>
                {fieldMappings.map((mapping, index) => (
                  <div key={index} className="flex items-center space-x-3">
                    <label className="text-sm font-medium text-gray-700 w-32">
                      Destination:
                    </label>
                    <select
                      value={mapping.destination}
                      onChange={(e) => updateFieldMapping(index, 'destination', e.target.value)}
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-primary-500 focus:border-primary-500"
                    >
                      <option value="">Select destination field...</option>
                      {destinationFields.map((field) => (
                        <option key={field} value={field}>
                          {field}
                        </option>
                      ))}
                    </select>
                    <span className="text-gray-500 text-xl">←</span>
                    <label className="text-sm font-medium text-gray-700 w-24">
                      Source:
                    </label>
                    <select
                      value={mapping.source}
                      onChange={(e) => updateFieldMapping(index, 'source', e.target.value)}
                      className="flex-1 px-3 py-2 border border-gray-300 rounded-lg focus:ring-primary-500 focus:border-primary-500"
                    >
                      <option value="">Select source field...</option>
                      {sourceFields.map((field) => (
                        <option key={field} value={field}>
                          {field}
                        </option>
                      ))}
                    </select>
                    <button
                      onClick={() => removeFieldMapping(index)}
                      className="px-3 py-2 text-red-600 hover:text-red-800 hover:bg-red-50 rounded-lg"
                    >
                      ×
                    </button>
                  </div>
                ))}
                <button
                  onClick={addFieldMapping}
                  className="w-full px-4 py-2 text-primary-600 border border-primary-600 rounded-lg hover:bg-primary-50 text-sm font-medium"
                >
                  + Add Mapping
                </button>
              </div>
            ) : (
              <div className="text-center py-8">
                {!sourceFields && !destinationFields && !sourceFieldsError && !destinationFieldsError ? (
                  <div className="text-gray-500">Loading fields...</div>
                ) : (
                  <div className="max-w-2xl mx-auto">
                    {(sourceFieldsError || destinationFieldsError) && (
                      <div className="bg-red-50 border border-red-200 rounded-lg p-6">
                        {renderError(sourceFieldsError || destinationFieldsError)}
                      </div>
                    )}
                    {!sourceFieldsError && !destinationFieldsError && sourceFields && destinationFields && sourceFields.length === 0 && destinationFields.length === 0 && (
                      <div className="text-gray-500">No fields available to map</div>
                    )}
                    <div className="mt-4">
                      <button
                        onClick={handleRetry}
                        disabled={isFetchingSourceFields || isFetchingDestinationFields}
                        className="px-6 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 font-medium transition-colors shadow-md disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {isFetchingSourceFields || isFetchingDestinationFields ? 'Retrying...' : '🔄 Retry'}
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        <div className="flex justify-end space-x-4 mt-8 pt-8 border-t">
          <button
            onClick={() => step > 1 ? setStep((step - 1) as Step) : navigate('/pipelines')}
            className="px-6 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50"
          >
            {step > 1 ? 'Back' : 'Cancel'}
          </button>
          <button
            onClick={handleNext}
            disabled={
              (step === 1 && !sourceDataObject) ||
              (step === 2 && (!sourceDataObject || !destinationDataObject)) ||
              (step === 3 && (!sourceDataObject || !destinationDataObject)) ||
              createPipelineMutation.isLoading
            }
            className="px-6 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {step < 3 ? 'Next' : 'Create Pipeline'}
          </button>
        </div>
      </div>
      <QuickBooksCredentialsModal
        isOpen={showQuickBooksModal}
        onClose={() => {
          setShowQuickBooksModal(false);
          setPendingQuickBooksApp(null);
        }}
        onSubmit={async (clientId, clientSecret) => {
          setShowQuickBooksModal(false);
          if (!pendingQuickBooksApp) return;

          try {
            const returnTo = pendingIsSource ? 'pipeline' : 'connections';
            const response = await fetch(
              `http://localhost:8080/api/oauth/quickbooks/initiate?user_id=temp&return_to=${returnTo}&app_id=${pendingQuickBooksApp.id}&company_id=${companyId}&client_id=${encodeURIComponent(clientId)}&client_secret=${encodeURIComponent(clientSecret)}`
            );

            if (!response.ok) {
              let errorMessage = `${response.status} ${response.statusText}`;
              try {
                const errorData = await response.json();
                if (errorData.error) {
                  errorMessage = errorData.error;
                }
              } catch (e) {
                const text = await response.text();
                if (text) {
                  errorMessage = text;
                }
              }
              console.error('QuickBooks OAuth initiate failed:', response.status, errorMessage);
              alert(`Failed to connect to QuickBooks: ${errorMessage}`);
              return;
            }

            const data = await response.json();
            if (data.auth_url) {
              // Store credentials in sessionStorage temporarily for callback
              sessionStorage.setItem('qb_client_id', clientId);
              sessionStorage.setItem('qb_client_secret', clientSecret);
              window.location.href = data.auth_url;
            } else {
              alert('QuickBooks OAuth endpoint did not return an auth_url');
            }
          } catch (error: any) {
            console.error('Failed to initiate QuickBooks OAuth:', error);
            alert(`Failed to connect to QuickBooks: ${error?.message || 'Unknown error'}`);
          }
        }}
      />
    </div>
  );
}
