import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from 'react-query';
import { pipelineApi } from '@domains/pipelines/adapters/pipelineApi';
import { Pipeline, SyncRun } from '@domains/pipelines/entities/Pipeline';
import { formatDistanceToNow } from 'date-fns';

type Tab = 'logs' | 'errors' | 'settings';

export function PipelineDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<Tab>('logs');

  const { data, isLoading } = useQuery<{ pipeline: Pipeline; status: any }>(
    ['pipeline', id],
    () => pipelineApi.getPipeline(id!),
    { enabled: !!id }
  );

  const { data: syncRuns, isLoading: isLoadingSyncRuns } = useQuery<SyncRun[]>(
    ['syncRuns', id],
    () => pipelineApi.getSyncRuns(id!),
    { 
      enabled: !!id,
      refetchInterval: 5000, // Refetch every 5 seconds to show latest sync runs
    }
  );

  const executeMutation = useMutation(
    () => pipelineApi.executePipeline(id!),
    {
      onSuccess: () => {
        // Show success message
        setTimeout(() => {
          queryClient.invalidateQueries({ queryKey: ['pipeline', id] });
          queryClient.invalidateQueries({ queryKey: ['syncRuns', id] });
          queryClient.invalidateQueries({ queryKey: ['pipelines'] });
        }, 1000); // Wait a bit for the sync to start
      },
      onError: (error: any) => {
        console.error('Failed to execute pipeline:', error);
        alert(error?.error || 'Failed to start sync. Please try again.');
      },
    }
  );

  const handleRunSync = () => {
    if (confirm('Start a manual sync for this pipeline?')) {
      executeMutation.mutate();
    }
  };

  if (isLoading || !data) {
    return <div className="text-center py-8">Loading...</div>;
  }

  const { pipeline } = data;
  const lastSyncTime = pipeline.checkpoint?.last_sync_time
    ? formatDistanceToNow(new Date(pipeline.checkpoint.last_sync_time), { addSuffix: true })
    : 'Never';
  
  // Calculate stats from real sync runs
  const successCount = syncRuns?.reduce((acc, run) => acc + run.records_written, 0) || 0;
  const failedCount = syncRuns?.reduce((acc, run) => {
    if (run.status === 'FAILED' || run.status === 'PARTIAL') {
      return acc + (run.records_read - run.records_written);
    }
    return acc;
  }, 0) || 0;
  
  // Get failed sync runs for Error Records tab
  const failedSyncRuns = syncRuns?.filter(run => run.status === 'FAILED' || (run.status === 'PARTIAL' && run.error)) || [];

  return (
    <div className="max-w-7xl mx-auto">
      <div className="mb-6">
        <button
          onClick={() => navigate('/pipelines')}
          className="text-primary-600 hover:text-primary-800 mb-4"
        >
          ← Back to Pipelines
        </button>
        <h1 className="text-3xl font-bold text-gray-900">
          Connection Details - {pipeline.source_object?.connection?.app?.display_name} to{' '}
          {pipeline.destination_object?.connection?.app?.display_name}
        </h1>
      </div>

      {/* Connection Summary */}
      <div className="bg-white rounded-lg shadow-lg p-6 mb-6">
        <div className="flex items-center justify-between">
          <div className="grid grid-cols-4 gap-6 flex-1">
            <div>
              <p className="text-sm text-gray-600">Last Sync</p>
              <p className="text-lg font-semibold text-gray-900">{lastSyncTime}</p>
            </div>
            <div>
              <p className="text-sm text-gray-600">Records Synced</p>
              <p className="text-lg font-semibold text-green-600 font-bold">{successCount}</p>
            </div>
            <div>
              <p className="text-sm text-gray-600">Failed Records</p>
              <p className="text-lg font-semibold text-red-600 font-bold">{failedCount}</p>
            </div>
            <div>
              <p className="text-sm text-gray-600">Sync Status</p>
              <p className={`text-lg font-semibold ${
                pipeline.status === 'ACTIVE' ? 'text-green-600' : 'text-yellow-600'
              }`}>
                {pipeline.status}
              </p>
            </div>
          </div>
          <button
            onClick={handleRunSync}
            disabled={executeMutation.isLoading}
            className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 font-medium shadow-md transition-colors flex items-center gap-2"
          >
            {executeMutation.isLoading ? (
              <>
                <span className="animate-spin">⏳</span>
                <span>Running...</span>
              </>
            ) : (
              <>
                <span>Run Sync</span>
                <span>{'>'}</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-white rounded-lg shadow-lg">
        <div className="border-b border-gray-200">
          <nav className="flex space-x-8 px-6">
            <button
              onClick={() => setActiveTab('logs')}
              className={`py-4 px-1 border-b-2 font-medium text-sm ${
                activeTab === 'logs'
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              Sync Logs
            </button>
            <button
              onClick={() => setActiveTab('errors')}
              className={`py-4 px-1 border-b-2 font-medium text-sm ${
                activeTab === 'errors'
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              Error Records
            </button>
            <button
              onClick={() => setActiveTab('settings')}
              className={`py-4 px-1 border-b-2 font-medium text-sm ${
                activeTab === 'settings'
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              Settings
            </button>
          </nav>
        </div>

        <div className="p-6">
          {activeTab === 'logs' && (
            <div>
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Sync Run ID
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Records Read
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Records Written
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Status
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                        Details
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {isLoadingSyncRuns ? (
                      <tr>
                        <td colSpan={5} className="px-6 py-4 text-center text-gray-500">
                          Loading sync logs...
                        </td>
                      </tr>
                    ) : syncRuns && syncRuns.length > 0 ? (
                      syncRuns.map((run) => (
                        <tr key={run.id} className="hover:bg-gray-50">
                          <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                            {run.id.substring(0, 8)}...
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-500">
                            {run.records_read} records read
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-500">
                            {run.records_written} records written
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <span className={`px-2 py-1 text-xs font-medium rounded ${
                              run.status === 'SUCCESS'
                                ? 'bg-green-100 text-green-800'
                                : run.status === 'FAILED'
                                ? 'bg-red-100 text-red-800'
                                : 'bg-yellow-100 text-yellow-800'
                            }`}>
                              {run.status}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm">
                            {run.error ? (
                              <div>
                                <p className="text-red-600 text-xs mb-1">Error: {run.error}</p>
                                <button className="text-primary-600 hover:text-primary-800">
                                  Details {'>'}
                                </button>
                              </div>
                            ) : (
                              <div>
                                <p className="text-xs text-gray-500 mb-1">
                                  Started: {formatDistanceToNow(new Date(run.started_at), { addSuffix: true })}
                                </p>
                                {run.finished_at && (
                                  <p className="text-xs text-gray-500">
                                    Finished: {formatDistanceToNow(new Date(run.finished_at), { addSuffix: true })}
                                  </p>
                                )}
                              </div>
                            )}
                          </td>
                        </tr>
                      ))
                    ) : (
                      <tr>
                        <td colSpan={5} className="px-6 py-4 text-center text-gray-500">
                          No sync runs yet. Click "Run Sync" to start a sync.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
              <div className="mt-4 text-center">
                <button className="text-primary-600 hover:text-primary-800">
                  View All Records {'>'}
                </button>
              </div>
            </div>
          )}

          {activeTab === 'errors' && (
            <div>
              {failedSyncRuns.length > 0 ? (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Sync Run ID
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Records Read
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Records Written
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Status
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Error Details
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Time
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {failedSyncRuns.map((run) => (
                        <tr key={run.id} className="hover:bg-gray-50">
                          <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                            {run.id.substring(0, 8)}...
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-500">
                            {run.records_read}
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-500">
                            {run.records_written}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <span className="px-2 py-1 text-xs font-medium bg-red-100 text-red-800 rounded">
                              {run.status}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-sm text-red-600">
                            {run.error || 'Unknown error'}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {formatDistanceToNow(new Date(run.started_at), { addSuffix: true })}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="text-center py-8 text-gray-500">
                  No error records found. All syncs completed successfully!
                </div>
              )}
            </div>
          )}

          {activeTab === 'settings' && (
            <div className="space-y-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Sync Frequency (minutes)
                </label>
                <input
                  type="number"
                  value={pipeline.schedule_interval}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg bg-gray-50"
                  readOnly={true}
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Status
                </label>
                <select
                  value={pipeline.status}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg"
                  disabled
                >
                  <option value="ACTIVE">Active</option>
                  <option value="PAUSED">Paused</option>
                </select>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

