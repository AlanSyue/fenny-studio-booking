export interface Course {
  id: number
  name: string
  image_path: string | null
  description: string | null
  product_url?: string | null
  woo_product_id?: number | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface Inquiry {
  id: number
  name: string | null
  email: string
  // Legacy single-course fields — kept for back-compat with older rows
  course_id?: number | null
  course_name?: string | null
  // New multi-course array
  courses: { id: number; name: string }[]
  headcount: string | null
  city?: string | null
  district?: string | null
  address?: string | null
  budget?: string | null
  additional_contact: string | null
  is_processed: boolean
  created_at: string
}

export interface LoginResponse {
  token: string
}
