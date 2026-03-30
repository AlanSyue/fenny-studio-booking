import { Link, useLocation } from 'react-router-dom'

interface Props {
  children: React.ReactNode
}

export default function AdminLayout({ children }: Props) {
  const location = useLocation()

  const handleLogout = () => {
    localStorage.removeItem('token')
    window.location.href = '/admin/login'
  }

  const navItems = [
    { path: '/admin/courses', label: '課程管理' },
    { path: '/admin/inquiries', label: '詢問列表' },
  ]

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex space-x-8">
              <div className="flex items-center">
                <span className="text-xl font-bold text-gray-900">Fenny Studio</span>
              </div>
              {navItems.map((item) => (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium ${
                    location.pathname === item.path
                      ? 'border-indigo-500 text-gray-900'
                      : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'
                  }`}
                >
                  {item.label}
                </Link>
              ))}
            </div>
            <div className="flex items-center">
              <button
                onClick={handleLogout}
                className="text-sm text-gray-500 hover:text-gray-700 cursor-pointer"
              >
                登出
              </button>
            </div>
          </div>
        </div>
      </nav>
      <main className="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
        {children}
      </main>
    </div>
  )
}
