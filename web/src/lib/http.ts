import axios from 'axios'
import { env } from '../config/env'

const http = axios.create({
  baseURL: env.apiBaseUrl,
  timeout: env.apiTimeoutMs,
  withCredentials: true,
})

export default http
