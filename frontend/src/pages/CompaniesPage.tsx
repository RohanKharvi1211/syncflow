import { useQuery } from 'react-query';
import { companyApi } from '@domains/companies/adapters/companyApi';
import { Company } from '@domains/companies/entities/Company';
import { useCompanyStore } from '@shared/store/companyStore';

export function CompaniesPage() {
  const { selectedCompany, setSelectedCompany } = useCompanyStore();

  const { data: companies, isLoading } = useQuery<Company[]>(
    'companies',
    () => companyApi.getCompanies()
  );

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-3xl font-bold text-gray-900">Companies</h1>
        <button className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700">
          + Add Company
        </button>
      </div>

      {isLoading ? (
        <div className="text-center py-8">Loading...</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {companies?.map((company) => (
            <div
              key={company.id}
              className={`bg-white rounded-lg shadow p-6 cursor-pointer border-2 transition ${
                selectedCompany?.id === company.id
                  ? 'border-primary-500'
                  : 'border-transparent hover:border-gray-200'
              }`}
              onClick={() => setSelectedCompany(company)}
            >
              <h3 className="text-lg font-semibold text-gray-900">{company.name}</h3>
              {company.domain && (
                <p className="text-sm text-gray-500 mt-1">{company.domain}</p>
              )}
              <div className="mt-4 flex items-center justify-between">
                <span
                  className={`px-2 py-1 rounded text-xs ${
                    company.is_active
                      ? 'bg-green-100 text-green-800'
                      : 'bg-gray-100 text-gray-800'
                  }`}
                >
                  {company.is_active ? 'Active' : 'Inactive'}
                </span>
                {selectedCompany?.id === company.id && (
                  <span className="text-primary-600 text-sm font-medium">Selected</span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}


