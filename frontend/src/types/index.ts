export interface Course {
  id: number
  name: string
  image_path: string | null
  description: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface Inquiry {
  id: number
  name: string | null
  email: string
  course_id: number | null
  course_name?: string | null
  headcount: string | null
  additional_contact: string | null
  is_processed: boolean
  created_at: string
}

export interface LoginResponse {
  token: string
}
