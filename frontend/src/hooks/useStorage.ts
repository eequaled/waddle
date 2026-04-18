import { useState, useEffect } from 'react';
import { GetSessions, GetAppDetails } from '../../wailsjs/go/main/App';
import { Session, AppDetail } from '../types';

export function useSessions() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const fetchSessions = async () => {
      try {
        const data = await GetSessions();
        // Assume GetSessions returns an array of objects that map reasonably to our Session type.
        setSessions((data as unknown) as Session[]);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setLoading(false);
      }
    };
    fetchSessions();
  }, []);

  return { sessions, loading, error };
}

export function useAppDetails(date: string) {
  const [appDetails, setAppDetails] = useState<AppDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!date) return;
    const fetchAppDetails = async () => {
      try {
        setLoading(true);
        const data = await GetAppDetails(date);
        setAppDetails((data as unknown) as AppDetail[]);
      } catch (err) {
        setError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setLoading(false);
      }
    };
    fetchAppDetails();
  }, [date]);

  return { appDetails, loading, error };
}
