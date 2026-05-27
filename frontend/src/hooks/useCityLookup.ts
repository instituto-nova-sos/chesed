import { useCallback, useState } from 'react';

export interface City {
  id: number;
  name: string;
}

export function useCityLookup() {
  const [cities, setCities] = useState<City[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const fetchCities = useCallback(async (stateCode: string) => {
    if (!stateCode) {
      setCities([]);
      return;
    }
    setIsLoading(true);
    try {
      const response = await fetch(
        `https://servicodados.ibge.gov.br/api/v1/localidades/estados/${stateCode}/municipios?orderBy=nome`,
      );
      if (!response.ok) {
        setCities([]);
        return;
      }
      const data: Array<{ id: number; nome: string }> = await response.json();
      setCities(data.map((m) => ({ id: m.id, name: m.nome })));
    } catch {
      setCities([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const clearCities = useCallback(() => {
    setCities([]);
  }, []);

  return { cities, fetchCities, clearCities, isLoading };
}
