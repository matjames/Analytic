SELECT f.*
FROM public.facilities f
JOIN public.facilities f2
  ON f.mfl_uid = f2.mfl_uid
 AND f.id > f2.id
ORDER BY f.mfl_uid, f.id;

DELETE FROM public.facilities f
USING public.facilities f2
WHERE f.mfl_uid = f2.mfl_uid
  AND f.id > f2.id;