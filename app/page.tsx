import Link from "next/link";

export default function Home() {
  return (
    <div className="min-h-screen flex flex-col items-center justify-center p-8">
      <main className="max-w-4xl w-full text-center space-y-8">
        <h1 className="text-6xl font-bold text-blue-600">SaaS K-ERP</h1>
        <p className="text-2xl text-gray-600 dark:text-gray-300">
          Enterprise Resource Planning System
        </p>
        
        <div className="grid md:grid-cols-2 gap-6 mt-12">
          <div className="border border-gray-300 dark:border-gray-700 rounded-lg p-6 hover:shadow-lg transition-shadow">
            <h2 className="text-xl font-semibold mb-3">📊 Dashboard</h2>
            <p className="text-gray-600 dark:text-gray-400 mb-4">
              View your business metrics and KPIs at a glance
            </p>
            <Link 
              href="/dashboard"
              className="inline-block bg-blue-600 text-white px-6 py-2 rounded hover:bg-blue-700 transition-colors"
            >
              Go to Dashboard
            </Link>
          </div>
          
          <div className="border border-gray-300 dark:border-gray-700 rounded-lg p-6 hover:shadow-lg transition-shadow">
            <h2 className="text-xl font-semibold mb-3">🔐 Authentication</h2>
            <p className="text-gray-600 dark:text-gray-400 mb-4">
              Secure login and user management system
            </p>
            <div className="inline-block bg-gray-300 dark:bg-gray-700 text-gray-600 dark:text-gray-400 px-6 py-2 rounded">
              Coming Soon
            </div>
          </div>
        </div>

        <div className="mt-12 pt-8 border-t border-gray-300 dark:border-gray-700">
          <h3 className="text-2xl font-semibold mb-6">Core ERP Modules</h3>
          <div className="grid md:grid-cols-3 gap-4 text-left">
            <div className="border border-gray-200 dark:border-gray-800 rounded p-4">
              <div className="text-2xl mb-2">📦</div>
              <h4 className="font-semibold">Inventory Management</h4>
              <p className="text-sm text-gray-600 dark:text-gray-400">Track stock levels and warehouse operations</p>
            </div>
            <div className="border border-gray-200 dark:border-gray-800 rounded p-4">
              <div className="text-2xl mb-2">💰</div>
              <h4 className="font-semibold">Sales & Orders</h4>
              <p className="text-sm text-gray-600 dark:text-gray-400">Manage sales orders and customer relationships</p>
            </div>
            <div className="border border-gray-200 dark:border-gray-800 rounded p-4">
              <div className="text-2xl mb-2">🛒</div>
              <h4 className="font-semibold">Purchasing</h4>
              <p className="text-sm text-gray-600 dark:text-gray-400">Handle purchase orders and supplier management</p>
            </div>
            <div className="border border-gray-200 dark:border-gray-800 rounded p-4">
              <div className="text-2xl mb-2">📈</div>
              <h4 className="font-semibold">Financial Management</h4>
              <p className="text-sm text-gray-600 dark:text-gray-400">Accounting, invoicing, and financial reporting</p>
            </div>
            <div className="border border-gray-200 dark:border-gray-800 rounded p-4">
              <div className="text-2xl mb-2">👥</div>
              <h4 className="font-semibold">HR Management</h4>
              <p className="text-sm text-gray-600 dark:text-gray-400">Employee records and payroll processing</p>
            </div>
            <div className="border border-gray-200 dark:border-gray-800 rounded p-4">
              <div className="text-2xl mb-2">📊</div>
              <h4 className="font-semibold">Reporting & Analytics</h4>
              <p className="text-sm text-gray-600 dark:text-gray-400">Business intelligence and data insights</p>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
