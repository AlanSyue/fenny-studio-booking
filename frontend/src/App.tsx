import { Routes, Route } from 'react-router-dom'
import InquiryForm from './pages/InquiryForm'
import AdminLogin from './pages/AdminLogin'
import CourseManagement from './pages/CourseManagement'
import InquiryList from './pages/InquiryList'
import ProtectedRoute from './components/ProtectedRoute'

function App() {
  return (
    <Routes>
      <Route path="/" element={<InquiryForm />} />
      <Route path="/admin/login" element={<AdminLogin />} />
      <Route
        path="/admin/courses"
        element={
          <ProtectedRoute>
            <CourseManagement />
          </ProtectedRoute>
        }
      />
      <Route
        path="/admin/inquiries"
        element={
          <ProtectedRoute>
            <InquiryList />
          </ProtectedRoute>
        }
      />
    </Routes>
  )
}

export default App
