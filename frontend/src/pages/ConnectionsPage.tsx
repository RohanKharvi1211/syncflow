import { useState } from 'react';
import { useQuery } from 'react-query';
import { useAuthStore } from '@shared/store/authStore';
import { useCompanyStore } from '@shared/store/companyStore';
import { connectionApi } from '@domains/connections/adapters/connectionApi';
import { Connection } from '@domains/connections/entities/Connection';
import { appApi } from '@domains/apps/adapters/appApi';
import { App } from '@domains/apps/entities/App';

export function ConnectionsPage() {
  const { user } = useAuthStore();
  const { selectedCompany } = useCompanyStore();
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedApp, setSelectedApp] = useState<App | null>(null);

  const { data: connections, isLoading } = useQuery<Connection[]>(
    ['connections', selectedCompany?.id],
    () => connectionApi.getConnections(selectedCompany?.id || user?.company_id || ''),
    {
      enabled: !!selectedCompany || !!user?.company_id,
    }
  );

  // Fetch source apps
  const { data: sourceApps } = useQuery<App[]>(
    ['apps', 'source'],
    () => appApi.getApps('source'),
    {
      enabled: showCreateModal,
    }
  );

  // Fetch destination apps
  const { data: destinationApps } = useQuery<App[]>(
    ['apps', 'destination'],
    () => appApi.getApps('destination'),
    {
      enabled: showCreateModal,
    }
  );

  const handleConnect = async (app: App) => {
    // For Google apps, initiate OAuth
    if (app.name === 'googlesheet' || app.name === 'googledrive') {
      try {
        const response = await fetch(`http://localhost:8080/api/oauth/google/initiate?user_id=temp`);
        const data = await response.json();
        if (data.auth_url) {
          window.location.href = data.auth_url;
        }
      } catch (error) {
        console.error('Failed to initiate OAuth:', error);
      }
    } else {
      // For other apps, show a message or handle differently
      alert(`Connection setup for ${app.display_name} is not yet implemented`);
    }
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold text-gray-900">Connections</h1>
        <button 
          onClick={() => setShowCreateModal(true)}
          className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
        >
          + Add Connection
        </button>
      </div>

      {isLoading ? (
        <div className="text-center py-8">Loading...</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {connections && connections.length > 0 ? (
            connections.map((connection) => (
              <div
                key={connection.id}
                className="bg-white rounded-lg shadow p-6 border-l-4 border-primary-500"
              >
                <div className="flex items-center justify-between mb-4">
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900">
                      {(() => {
                        // For Google Sheets/Drive, show "Google Sheet - SheetName" format
                        if (connection.app?.name === 'googlesheet' || connection.app?.name === 'googledrive') {
                          if (connection.data_objects && connection.data_objects.length > 0) {
                            const dataObject = connection.data_objects[0];
                            let config = dataObject.config;
                            
                            // Handle case where config might be a JSON string
                            if (typeof config === 'string') {
                              try {
                                config = JSON.parse(config);
                              } catch (e) {
                                // If parsing fails, use identifier as fallback
                                return `Google Sheet - ${dataObject.identifier}`;
                              }
                            }
                            
                            const sheetName = config?.file_name || 
                                             config?.fileName ||
                                             dataObject.identifier;
                            return `Google Sheet - ${sheetName}`;
                          }
                          return 'Google Sheet';
                        }
                        // For other apps, show the app display name
                        return connection.app?.display_name || connection.type || 'Unknown';
                      })()}
                    </h3>
                  </div>
                  <span
                    className={`px-2 py-1 rounded text-xs ${
                      connection.status === 'ACTIVE'
                        ? 'bg-green-100 text-green-800'
                        : connection.status === 'EXPIRED'
                        ? 'bg-yellow-100 text-yellow-800'
                        : 'bg-red-100 text-red-800'
                    }`}
                  >
                    {connection.status}
                  </span>
                </div>
                <div className="space-y-2 text-sm text-gray-600">
                  <p>Type: {connection.auth_type}</p>
                  {connection.token_expires_at && (
                    <p>
                      Expires: {new Date(connection.token_expires_at).toLocaleDateString()}
                    </p>
                  )}
                </div>
                <div className="mt-4 flex space-x-2">
                  <button className="flex-1 px-3 py-2 text-sm border rounded-lg hover:bg-gray-50">
                    Edit
                  </button>
                  <button className="flex-1 px-3 py-2 text-sm border rounded-lg hover:bg-gray-50">
                    Reconnect
                  </button>
                </div>
              </div>
            ))
          ) : (
            <div className="col-span-3 text-center py-8 text-gray-500">
              No connections found
            </div>
          )}
        </div>
      )}

      {/* Create Connection Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
            <div className="p-6">
              <div className="flex justify-between items-center mb-6">
                <h2 className="text-2xl font-bold text-gray-900">Create New Connection</h2>
                <button
                  onClick={() => {
                    setShowCreateModal(false);
                    setSelectedApp(null);
                  }}
                  className="text-gray-400 hover:text-gray-600"
                >
                  ✕
                </button>
              </div>

              <div className="space-y-6">
                {/* Source Apps Section */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-3">
                    Select Source App
                  </label>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {sourceApps?.map((app) => (
                      <div
                        key={app.id}
                        onClick={() => setSelectedApp(app)}
                        className={`p-4 border-2 rounded-lg cursor-pointer transition-all ${
                          selectedApp?.id === app.id
                            ? 'border-primary-500 bg-primary-50'
                            : 'border-gray-200 hover:border-gray-300'
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <div>
                            <h3 className="font-semibold text-gray-900">{app.display_name}</h3>
                            {app.description && (
                              <p className="text-sm text-gray-600 mt-1">{app.description}</p>
                            )}
                          </div>
                          {app.name === 'googlesheet' && <span className="text-2xl">📊</span>}
                          {app.name === 'googledrive' && <span className="text-2xl">📁</span>}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Destination Apps Section */}
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-3">
                    Select Destination App
                  </label>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {destinationApps?.map((app) => (
                      <div
                        key={app.id}
                        onClick={() => setSelectedApp(app)}
                        className={`p-4 border-2 rounded-lg cursor-pointer transition-all ${
                          selectedApp?.id === app.id
                            ? 'border-primary-500 bg-primary-50'
                            : 'border-gray-200 hover:border-gray-300'
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <div>
                            <h3 className="font-semibold text-gray-900">{app.display_name}</h3>
                            {app.description && (
                              <p className="text-sm text-gray-600 mt-1">{app.description}</p>
                            )}
                          </div>
                          {app.name === 'quickbooks' && <span className="text-2xl">💰</span>}
                          {app.name === 'tally' && <span className="text-2xl">📊</span>}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Connect Button */}
                {selectedApp && (
                  <div className="flex justify-end space-x-3 pt-4 border-t">
                    <button
                      onClick={() => {
                        setShowCreateModal(false);
                        setSelectedApp(null);
                      }}
                      className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={() => handleConnect(selectedApp)}
                      className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
                    >
                      Connect {selectedApp.display_name}
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

