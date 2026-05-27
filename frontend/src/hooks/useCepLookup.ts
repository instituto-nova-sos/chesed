import { useCallback, useState } from 'react';

export interface CepResult {
  street: string;
  neighborhood: string;
  city: string;
  state: string;
}

interface ViaCepResponse {
  logradouro?: string;
  bairro?: string;
  localidade?: string;
  uf?: string;
  erro?: boolean;
}

interface BrasilApiResponse {
  street?: string;
  neighborhood?: string;
  city?: string;
  state?: string;
}

async function fetchWithRetry(
  url: string,
  retries: number = 1,
): Promise<Response> {
  const response = await fetch(url);
  if (response.status >= 500 && retries > 0) {
    await new Promise((r) => setTimeout(r, 1000));
    return fetchWithRetry(url, retries - 1);
  }
  return response;
}

async function tryViaCep(cep: string): Promise<CepResult | null> {
  try {
    const response = await fetchWithRetry(
      `https://viacep.com.br/ws/${cep}/json/`,
    );
    if (!response.ok) return null;
    const data: ViaCepResponse = await response.json();
    if (data.erro) return null;
    return {
      street: data.logradouro || '',
      neighborhood: data.bairro || '',
      city: data.localidade || '',
      state: data.uf || '',
    };
  } catch {
    return null;
  }
}

async function tryBrasilApi(cep: string): Promise<CepResult | null> {
  try {
    const response = await fetchWithRetry(
      `https://brasilapi.com.br/api/cep/v2/${cep}`,
    );
    if (!response.ok) return null;
    const data: BrasilApiResponse = await response.json();
    return {
      street: data.street || '',
      neighborhood: data.neighborhood || '',
      city: data.city || '',
      state: data.state || '',
    };
  } catch {
    return null;
  }
}

export function useCepLookup() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const lookupCep = useCallback(
    async (cep: string): Promise<CepResult | null> => {
      const digits = cep.replace(/\D/g, '');
      if (digits.length !== 8) return null;

      setIsLoading(true);
      setError(null);

      try {
        // Try ViaCEP first
        const viaCepResult = await tryViaCep(digits);
        if (viaCepResult) return viaCepResult;

        // Fallback to BrasilAPI
        const brasilApiResult = await tryBrasilApi(digits);
        if (brasilApiResult) return brasilApiResult;

        setError('CEP não encontrado');
        return null;
      } catch {
        setError('Erro ao buscar CEP');
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  return { lookupCep, isLoading, error };
}
