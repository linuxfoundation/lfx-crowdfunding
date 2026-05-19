// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import { type QueryFunction, useInfiniteQuery } from '@tanstack/vue-query';
import { type MaybeRef, computed, toValue } from 'vue';
import type { InitiativeBase } from '#shared/types/initiative.types';
import type { Pagination } from '#shared/types/pagination';
function getNextInitiativesPageParam(lastPage: Pagination<InitiativeBase>) {
  const totalPages = Math.ceil(lastPage.total / lastPage.pageSize);
  return lastPage.page < totalPages ? lastPage.page + 1 : null;
}

function fetchInitiativesQueryFn(
  query: () => Record<string, string>,
): QueryFunction<Pagination<InitiativeBase>, readonly unknown[], number> {
  return async ({ pageParam = 1 }) => {
    const params = new URLSearchParams({ ...query(), page: String(pageParam) });
    return $fetch<Pagination<InitiativeBase>>(`/api/initiatives?${params}`);
  };
}

export function useInitiatives(params?: {
  search?: MaybeRef<string>;
  type?: MaybeRef<string>;
  sort?: MaybeRef<string>;
  pageSize?: MaybeRef<number>;
}) {
  const resolvedParams = computed(() => ({
    search: toValue(params?.search ?? ''),
    type: toValue(params?.type ?? ''),
    sort: toValue(params?.sort ?? ''),
    pageSize: toValue(params?.pageSize ?? 0),
  }));

  const queryKey = computed(() => ['initiatives', ...Object.values(resolvedParams.value)]);

  const queryFn = fetchInitiativesQueryFn(() => {
    const { search, type, sort, pageSize } = resolvedParams.value;
    const p: Record<string, string> = {};
    if (search) p.search = search;
    if (type) p.type = type;
    if (sort) p.sort = sort;
    if (pageSize) p.pageSize = String(pageSize);
    return p;
  });

  const result = useInfiniteQuery<
    Pagination<InitiativeBase>,
    Error,
    Pagination<InitiativeBase>,
    readonly unknown[],
    number
  >({
    queryKey,
    queryFn,
    getNextPageParam: getNextInitiativesPageParam,
    initialPageParam: 1,
  });

  return result;
}
