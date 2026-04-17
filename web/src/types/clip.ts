export interface Clip {
  id: string;
  source_id: string;
  start_time: string;
  end_time: string;
  headline: string;
  s3_path: string;
  status: 'pending' | 'approved' | 'rejected';
  viral_score: number;
}
