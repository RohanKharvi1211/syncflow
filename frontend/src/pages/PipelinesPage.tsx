import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from 'react-query';
import { useAuthStore } from '@shared/store/authStore';
import { useCompanyStore } from '@shared/store/companyStore';
import { pipelineApi } from '@domains/pipelines/adapters/pipelineApi';
import { Pipeline } from '@domains/pipelines/entities/Pipeline';

export function PipelinesPage() {
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const { selectedCompany } = useCompanyStore();
  const queryClient = useQueryClient();
  const [selectedPipeline, setSelectedPipeline] = useState<Pipeline | null>(null);

  const { data: pipelines, isLoading } = useQuery<Pipeline[]>(
    ['pipelines', selectedCompany?.id],
    () => pipelineApi.getPipelines(selectedCompany?.id || user?.company_id || ''),
    {
      enabled: !!selectedCompany || !!user?.company_id,
    }
  );

  const executeMutation = useMutation(
    (id: string) => pipelineApi.executePipeline(id),
    {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['pipelines'] });
      },
    }
  );

  const handleExecute = (id: string) => {
    if (confirm('Execute this pipeline now?')) {
      executeMutation.mutate(id);
    }
  };

  if (isLoading) {
    return <div className="text-center py-8">Loading...</div>;
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold text-gray-900">Pipelines</h1>
        <button
          onClick={() => navigate('/pipelines/create')}
          className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
        >
          + Create Pipeline
        </button>
      </div>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Source → Destination
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Type
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Schedule
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Status
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Last Sync
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {pipelines && pipelines.length > 0 ? (
              pipelines.map((pipeline) => (
                <tr key={pipeline.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-gray-900">
                      {pipeline.source_object?.connection?.app?.display_name || 'Unknown'} →{' '}
                      {pipeline.destination_object?.connection?.app?.display_name || 'Unknown'}
                    </div>
                    <div className="text-sm text-gray-500">
                      {pipeline.source_object?.identifier} → {pipeline.destination_object?.identifier}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {pipeline.sync_type}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    Every {pipeline.schedule_interval} min
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                        pipeline.status === 'ACTIVE'
                          ? 'bg-green-100 text-green-800'
                          : 'bg-yellow-100 text-yellow-800'
                      }`}
                    >
                      {pipeline.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {pipeline.checkpoint?.last_sync_time
                      ? new Date(pipeline.checkpoint.last_sync_time).toLocaleString()
                      : 'Never'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button
                      onClick={() => navigate(`/pipelines/${pipeline.id}`)}
                      className="text-primary-600 hover:text-primary-900 mr-4"
                    >
                      View Details
                    </button>
                    <button
                      onClick={() => handleExecute(pipeline.id)}
                      className="text-primary-600 hover:text-primary-900 mr-4"
                    >
                      Execute
                    </button>
                    <button className="text-gray-600 hover:text-gray-900">Edit</button>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={6} className="px-6 py-4 text-center text-gray-500">
                  No pipelines found
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

